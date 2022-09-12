package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/minepkg/minepkg/internals/auth"
	"github.com/minepkg/minepkg/internals/instances"
	"github.com/minepkg/minepkg/internals/launcher"
	"github.com/minepkg/minepkg/internals/remote"
	"github.com/minepkg/minepkg/pkg/manifest"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(protoHandlerCmd)
}

const prefixLength = len("minepkg://")

var (
	StatusIdle           = "idle"
	StatusStarting       = "starting"
	StatusRunningLoading = "running:loading"
	StatusRunningReady   = "running:ready"
)

var protoHandlerCmd = &cobra.Command{
	Use:    "proto-handler",
	Args:   cobra.ExactArgs(1),
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		realArg := args[0][prefixLength:]
		parsed := strings.Split(realArg, "/")
		action := parsed[0]

		switch action {
		case "launch":
			if len(parsed) != 2 {
				panic("Invalid launch command")
			}
			protoLaunch(parsed[1])
		}
	},
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

type ProgressEvent struct {
	Progress float32 `json:"progress"`
	Message  string  `json:"message"`
}

type CrashEvent struct {
	Message string `json:"message"`
	Error   error  `json:"error"`
	Stack   string `json:"stack"`
}

type InitialStatusEvent struct {
	Status string `json:"status"`
	Logs   string `json:"logs"`
}

type GameLogEvent struct {
	Log string `json:"log"`
	Tag string `json:"tag,omitempty"`
}

type LogForwarder struct {
	TheThing *TheThing
	tag      string
}

type StatsState struct {
	Memory []float32 `json:"memory"`
	CPU    []float32 `json:"cpu"`
}

type LocalState struct {
	Status string     `json:"status"`
	Stats  StatsState `json:"stats"`
}

type StateResponse struct {
	Status   string             `json:"status,omitempty"`
	Stats    StatsState         `json:"stats,omitempty"`
	Manifest *manifest.Manifest `json:"manifest,omitempty"`
	Logs     []*GameLogEvent    `json:"logs,omitempty"`
}

type TheThing struct {
	*remote.Connection
	State      *LocalState
	launcher   *launcher.Launcher
	logsBuffer []*GameLogEvent
}

// writes log stores the last 100 lines of logs in `logsBuffer`
func (t *TheThing) WriteLog(log *GameLogEvent) {

	// check readyness
	if strings.Contains(log.Log, "Sound engine started") {
		t.State.Status = StatusRunningReady
		// send status update
		t.Send("State", &StateResponse{Status: t.State.Status})
	}

	t.logsBuffer = append(t.logsBuffer, log)
	if len(t.logsBuffer) > 100 {
		t.logsBuffer = t.logsBuffer[1:]
	}
	t.Send("GameLog", log)
}

func (t *TheThing) Launch(man *manifest.Manifest) error {
	connection := t.Connection
	t.State.Status = StatusStarting

	if root.authProvider == nil {
		root.restoreAuth()
	}

	switch root.authProvider.(type) {
	case *auth.Microsoft:
		fmt.Println("Microsoft auth provider detected")
	case *auth.Mojang:
		fmt.Println("Mojang auth provider detected")
	default:
		fmt.Println("No auth provider detected")
		// Trigger Microsoft auth provider
		connection.Send("GameAuthRequired", nil)
		log.Println("Waiting for auth!")

		var waitChan chan error

		connection.HandleFunc("GameAuthAction", func(req *remote.Message) *remote.Response {
			waitChan <- nil
			return nil
		})

		// TODO: timeout
		<-waitChan
		root.useMicrosoftAuth()
		if err := root.authProvider.Prompt(); err != nil {
			panic(err)
		}
	}

	connection.Send("GameAuthenticated", nil)

	instance, err := newInstanceFromManifest(man)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", instance.Manifest.Package.Name)

	t.launcher = &launcher.Launcher{
		Instance: instance,
	}

	ctx := context.Background()

	connection.Send("progress", &ProgressEvent{
		Progress: 0.15,
		Message:  "Preparing Requirements",
	})

	// update requirements if needed
	outdatedReqs, err := t.launcher.PrepareRequirements()
	if err != nil {
		panic(err)
		// return fmt.Errorf("failed to update requirements: %w", err)
	}

	connection.Send("progress", &ProgressEvent{
		Progress: 0.30,
		Message:  "Preparing Minecraft",
	})

	// download minecraft (assets, libraries, main jar etc) if needed
	// needs to happen before javaUpdate because launch manifest
	// might contain wanted java version
	if err := t.launcher.PrepareMinecraft(ctx); err != nil {
		panic(err)
		// return fmt.Errorf("failed to download minecraft: %w", err)
	}

	// update java in the background if needed
	javaUpdate := t.launcher.PrepareJavaBg(ctx)

	connection.Send("progress", &ProgressEvent{
		Progress: 0.45,
		Message:  "Preparing Mods",
	})

	// update dependencies
	if err := t.launcher.PrepareDependencies(ctx, outdatedReqs); err != nil {
		panic(err)
	}

	connection.Send("progress", &ProgressEvent{
		Progress: 0.50,
		Message:  "Preparing for launch",
	})

	if err := instance.CopyLocalSaves(); err != nil {
		panic(err)
	}

	if err := instance.EnsureDependencies(ctx); err != nil {
		panic(err)
	}

	if err := instance.CopyOverwrites(); err != nil {
		panic(err)
	}

	if err := <-javaUpdate; err != nil {
		panic(err)
	}

	connection.Send("progress", &ProgressEvent{
		Progress: 0.5,
		Message:  "Starting Minecraft …",
	})

	collector := statsCollector{Thing: t, stop: make(chan struct{})}
	forwarder := &LogForwarder{TheThing: t, tag: "internal"}
	forwarderStdout := &LogForwarder{TheThing: t, tag: "game/stdout"}
	forwarderStderr := &LogForwarder{TheThing: t, tag: "game/stderr"}
	forwarder.Write([]byte("[LOG] Starting logger\n"))

	go func() {
		fmt.Printf("savegame: %s", instance.Manifest.Package.Savegame)
		t.State.Status = StatusRunningLoading
		t.launcher.Run(&instances.LaunchOptions{
			Stdout: forwarderStdout,
			Stderr: forwarderStderr,
			Env: []string{
				"MINEPKG_COMPANION_START_MINIMIZED=1",
			},
			StartSave: instance.Manifest.Package.Savegame,
		})
		log.Println("Minecraft was stopped")
		t.State.Status = StatusIdle
		connection.Send("GameStopped", nil)
		collector.Stop()
		time.Sleep(1 * time.Second)
	}()

	go func() {
		// TODO: remove this hack
		time.Sleep(time.Millisecond * 100)
		collector.Watch(t.launcher.Cmd.Process)
	}()

	runtime.GC()

	// TODO: figure out when minecraft is ready
	return nil
}

func (t *TheThing) Stop() error {
	if t.launcher == nil {
		return errors.New("no launcher running")
	}

	waitChan := make(chan error)
	go func() {
		waitChan <- t.launcher.Cmd.Wait()
	}()
	// signal stop now, wait for exit and kill if unresponsive for 5 seconds
	go t.launcher.Cmd.Process.Signal(syscall.SIGINT)
	select {
	case <-waitChan:
		break
	case <-time.After(5 * time.Second):
		log.Println("Minecraft did not exit, stopping forcefully")
		t.launcher.Cmd.Process.Kill()
	}

	return <-waitChan
}

func (f *LogForwarder) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	fmt.Print(string(p))
	f.TheThing.WriteLog(&GameLogEvent{Log: string(p), Tag: f.tag})
	return len(p), nil
}

func protoLaunch(pack string) {
	log.Println("Launching via protocol: " + pack)
	// connect to web client
	connection := remote.New()
	log.Println("Wait for handshake")

	theThing := &TheThing{
		Connection: connection,
		State:      &LocalState{Stats: StatsState{}, Status: StatusIdle},
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		log.Println("Interrupt received, shutting down")
		connection.Stop()
	}()

	connection.HandleFunc("requestState", func(event *remote.Message) *remote.Response {
		log.Println("sending game state")

		state := &StateResponse{
			Status: theThing.State.Status,
			Stats:  theThing.State.Stats,
			Logs:   theThing.logsBuffer,
		}

		if theThing.launcher != nil {
			state.Manifest = theThing.launcher.Instance.Manifest
		}

		return &remote.Response{
			Data: state,
		}
	})

	connection.HandleFunc("launch", func(event *remote.Message) *remote.Response {
		log.Println("============== LAUNCH ===============")
		// get data as manifest
		var man manifest.Manifest
		if err := json.Unmarshal(event.Data, &man); err != nil {
			panic(err)
		}
		theThing.Launch(&man)

		return &remote.Response{Data: theThing.State}
	})

	connection.HandleFunc("stop", func(event *remote.Message) *remote.Response {
		log.Println("============== STOP STOP STOP STOP ===============")
		if err := theThing.Stop(); err != nil {
			// TODO: return error, not nil
			log.Println("failed to stop:", err)
			return nil
		}

		return &remote.Response{Data: theThing.State}
	})

	mpkgLogForwarder := LogForwarder{TheThing: theThing, tag: "internal/log"}
	// mpkgLogForwarder.Write([]byte("Starting minepkg logger\n"))
	log.SetOutput(&mpkgLogForwarder)

	// log.Println("Handshake complete")

	// recovery handler, we need this because there is no other output
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Recovering from panic:", err)
			// send crash progress
			connection.Send("progress", &ProgressEvent{
				Progress: 0,
				Message:  "minepkg crashed!",
			})
			// build crash event
			crashEvent := &CrashEvent{
				Message: "minepkg crashed!",
			}
			if err.(error) != nil {
				crashEvent.Error = err.(error)
				crashEvent.Stack = string(debug.Stack())
				// try to get stacktrace from error directly
				if err, ok := err.(stackTracer); ok {
					for _, f := range err.StackTrace() {
						crashEvent.Stack += fmt.Sprintf("%+s:%d\n", f, f)
					}
				}
			}
			connection.Send("ClientCrash", crashEvent)
			// wait until event is sent (kinda hacky)
			time.Sleep(time.Second * 1)
			// output for local debugging
			fmt.Println("Client crashed!")
			fmt.Println(crashEvent.Error)
			fmt.Println(crashEvent.Stack)
			os.Exit(1)
		}
	}()

	// serve
	log.Println("serving")
	connection.ListenAndServe()

	fmt.Println("DONE")
	os.Exit(0)

}

type statsCollector struct {
	stop  chan struct{}
	Thing *TheThing
}

type StatsEvent struct {
	Memory               *mem.VirtualMemoryStat `json:"memory"`
	ProcessMemoryPercent float32                `json:"processMemoryPercent"`
	ProcessMemoryMiB     float32                `json:"processMemoryMiB"`
	ProcessCPUPercent    float64                `json:"processCPUPercent"`
}

func (s *statsCollector) Watch(p *os.Process) {
	monitor, err := process.NewProcess(int32(p.Pid))
	if err != nil {
		panic(err)
	}

	collect := func() {
		v, _ := mem.VirtualMemory()
		memP, _ := monitor.MemoryPercent()
		memInfo, _ := monitor.MemoryInfo()
		// cpuP, e := monitor.CPUPercent()
		// cpuT, _ := monitor.Times()
		cpuWAT, _ := monitor.Percent(time.Millisecond * 1000)
		// cpuG, _ := cpu.Times(false)

		// should probably check errors, but eh
		if memInfo == nil {
			return
		}

		s.Thing.State.Stats.Memory = append(s.Thing.State.Stats.Memory, float32(memInfo.RSS/1024/1024))
		s.Thing.State.Stats.CPU = append(s.Thing.State.Stats.CPU, float32(cpuWAT))

		s.Thing.Send("GameStats", &StatsEvent{
			Memory:               v,
			ProcessMemoryPercent: memP,
			ProcessMemoryMiB:     float32(memInfo.RSS / 1024 / 1024),
			ProcessCPUPercent:    cpuWAT,
		})
	}

	for {
		select {
		case <-s.stop:
			return
		default:
			time.Sleep(time.Millisecond * 1000)
			collect()
		}
	}
}

func (s *statsCollector) Stop() {
	close(s.stop)
}

func newInstanceFromManifest(release *manifest.Manifest) (*instances.Instance, error) {
	// set instance details
	instance := instances.New()
	instance.ProviderStore = root.ProviderStore
	instance.Manifest = manifest.NewInstanceLike(release)
	instance.Directory = filepath.Join(instance.InstancesDir(), release.Package.Name+"_"+release.Package.Platform)

	creds, err := root.getLaunchCredentialsOrLogin()
	if err != nil {
		return nil, err
	}
	instance.SetLaunchCredentials(creds)

	return instance, nil
}

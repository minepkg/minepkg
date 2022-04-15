package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/internals/auth"
	"github.com/minepkg/minepkg/internals/globals"
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

type LogForwarder struct {
	Remote *remote.Connection
}

func (f *LogForwarder) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	// log.Println("[LOG]", string(p))
	f.Remote.SendEvent("GameLog", string(p))
	return len(p), nil
}

func protoLaunch(pack string) {
	log.Println("Launching via protocol: " + pack)
	// connect to web client
	connection := remote.New()
	log.Println("Wait for handshake")
	// context with 5m timeout
	handshakeCtx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()
	if err := connection.ListenForHandshake(handshakeCtx); err != nil {
		// this is very bad, but we can't do anything about it
		// TODO: write error to logfile
		panic(err)
	}

	mpkgLogForwarder := LogForwarder{Remote: connection}
	mpkgLogForwarder.Write([]byte("Starting minepkg logger\n"))
	log.SetOutput(&mpkgLogForwarder)

	log.Println("Handshake complete")

	// recovery handler, we need this because there is no other output
	defer func() {
		if err := recover(); err != nil {
			// send crash progress
			connection.SendEvent("progress", &ProgressEvent{
				Progress: 0,
				Message:  "Client crashed!",
			})
			// build crash event
			crashEvent := &CrashEvent{
				Message: "Client crashed!",
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
			connection.SendEvent("ClientCrash", crashEvent)
			// wait until event is sent (kinda hacky)
			time.Sleep(time.Second * 1)
			// output for local debugging
			fmt.Println("Client crashed!")
			fmt.Println(crashEvent.Error)
			fmt.Println(crashEvent.Stack)
			os.Exit(1)
		}
	}()

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
		connection.SendEvent("GameAuthRequired", nil)
		log.Println("Waiting for auth!")

		timeoutContext, cancel := context.WithTimeout(context.Background(), time.Minute*5)
		defer cancel()
		connection.WaitFor(timeoutContext, "GameAuthAction")
		root.useMicrosoftAuth()
		if err := root.authProvider.Prompt(); err != nil {
			panic(err)
		}
	}

	connection.SendEvent("GameAuthenticated", nil)

	err := connection.SendEvent("progress", &ProgressEvent{
		Progress: 0.10,
		Message:  "Querying minepkg for " + pack,
	})
	if err != nil {
		panic(err)
	}

	release, err := findLatestRelease(pack)
	if err != nil {
		panic(err)
	}
	err = connection.SendEvent("progress", &ProgressEvent{
		Progress: 0.20,
		Message:  "Setting up instance for " + release.Package.Name,
	})
	if err != nil {
		panic(err)
	}

	instance, err := newInstanceFromRelease(release)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", instance.Manifest.Package.Name)

	launch := launcher.Launcher{
		Instance: instance,
	}

	ctx := context.Background()

	connection.SendEvent("progress", &ProgressEvent{
		Progress: 0.30,
		Message:  "Preparing Requirements",
	})

	// update requirements if needed
	outdatedReqs, err := launch.PrepareRequirements()
	if err != nil {
		panic(err)
		// return fmt.Errorf("failed to update requirements: %w", err)
	}

	connection.SendEvent("progress", &ProgressEvent{
		Progress: 0.60,
		Message:  "Preparing Minecraft",
	})

	// download minecraft (assets, libraries, main jar etc) if needed
	// needs to happen before javaUpdate because launch manifest
	// might contain wanted java version
	if err := launch.PrepareMinecraft(ctx); err != nil {
		panic(err)
		// return fmt.Errorf("failed to download minecraft: %w", err)
	}

	// update java in the background if needed
	javaUpdate := launch.PrepareJavaBg(ctx)

	connection.SendEvent("progress", &ProgressEvent{
		Progress: 0.80,
		Message:  "Preparing Mods",
	})

	// update dependencies
	if err := launch.PrepareDependencies(ctx, outdatedReqs); err != nil {
		panic(err)
	}

	connection.SendEvent("progress", &ProgressEvent{
		Progress: 0.90,
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

	connection.SendEvent("progress", &ProgressEvent{
		Progress: 1,
		Message:  "Launching …",
	})

	event := connection.ReceiveChannel()

	collector := statsCollector{Remote: connection, stop: make(chan struct{})}
	forwarder := &LogForwarder{Remote: connection}
	forwarder.Write([]byte("[LOG] Starting logger\n"))

	launchErr := make(chan error)
	go func() {
		fmt.Printf("savegame: %s", instance.Manifest.Package.Savegame)
		launchErr <- launch.Run(&instances.LaunchOptions{
			Stdout: forwarder,
			Stderr: forwarder,
			Env: []string{
				"MINEPKG_COMPANION_START_MINIMIZED=1",
			},
			StartSave: instance.Manifest.Package.Savegame,
		})
	}()

	go func() {
		// TODO: remove this hack
		time.Sleep(time.Millisecond * 100)
		collector.Watch(launch.Cmd.Process)
	}()

	defer connection.Close()
	defer func() {
		connection.SendEvent("GameStopped", nil)
		collector.Stop()
		time.Sleep(1 * time.Second)
	}()

	runtime.GC()

	for {
		select {
		case event := <-event:
			fmt.Println(event)
			switch event.Event {
			// case "focus":
			// 	launch.Cmd.Process.Signal(syscall.SIGUSR1)
			// case "pause":
			// 	launch.Cmd.Process.Signal(syscall.SIGSTOP)
			// 	connection.SendEvent("GamePaused", nil)
			// case "resume":
			// 	launch.Cmd.Process.Signal(syscall.SIGCONT)
			// 	connection.SendEvent("GameResumed", nil)
			case "stop":
				waitChan := make(chan error)
				go func() {
					waitChan <- launch.Cmd.Wait()
				}()
				// signal stop now, wait for exit and kill if unresponsive for 5 seconds
				go launch.Cmd.Process.Signal(syscall.SIGINT)
				select {
				case <-waitChan:
					break
				case <-time.After(5 * time.Second):
					log.Println("Minecraft did not exit, stopping forcefully")
					launch.Cmd.Process.Kill()
				}
				return
			}
		case err := <-launchErr:
			if err != nil {
				panic(err)
			}
			return
		}
	}
}

type statsCollector struct {
	stop   chan struct{}
	Remote *remote.Connection
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

		s.Remote.SendEvent("GameStats", &StatsEvent{
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

func findLatestRelease(name string) (*api.Release, error) {
	parts := strings.Split(name, "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid release specifier: %s", name)
	}
	query := &api.ReleasesQuery{
		Name:         parts[0],
		Platform:     "fabric",
		VersionRange: parts[1],
	}

	release, err := globals.ApiClient.ReleasesQuery(context.TODO(), query)
	if err != nil {
		return nil, err
	}

	if release.Package.Type == "mod" {
		return nil, errCanOnlyLaunchModpacks
	}

	return release, nil
}

func newInstanceFromRelease(release *api.Release) (*instances.Instance, error) {
	// set instance details
	instance := instances.New()
	instance.Manifest = manifest.NewInstanceLike(release.Manifest)
	instance.Directory = filepath.Join(instance.InstancesDir(), release.Package.Name+"_"+release.Package.Platform)

	creds, err := root.getLaunchCredentialsOrLogin()
	if err != nil {
		return nil, err
	}
	instance.SetLaunchCredentials(creds)

	return instance, nil
}

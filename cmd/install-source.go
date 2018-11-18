package cmd

import (
	"time"
	"github.com/briandowns/spinner"
	"io/ioutil"
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fiws/minepkg/internals/instances"
	"github.com/manifoldco/promptui"
	"github.com/logrusorgru/aurora"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

var (
	// errNoJarFile is returned when there where no jar files to extract
	errNoJarFile = errors.New("Did not find any jar files")
)

func installFromSource(url string, instance *instances.McInstance) {

	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	_, err = cli.Info(ctx); if err != nil {
		// TODO: maybe ask if we should install it?
		logger.Fail("You will need to install and start docker to install mods from source.")
	}

	logger.Info("Installing from source. This might take some time and you have to manage possible dependencies by yourself.")
	prompt := promptui.Prompt{
		Label: "Do you want want to continue?",
		IsConfirm: true,
		Default:   "Y",
	}

	_, err = prompt.Run()
	if err != nil {
		logger.Info("Aborting installation")
		os.Exit(0)
	}

	// TODO: check date! update old gradle images
	if history, err := cli.ImageHistory(ctx, "gradle"); err != nil || len(history) == 0 {
		logger.Info("Downloading gradle image (this might take a very long time)")

		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond) // Build our new spinner
		s.Prefix = "  "
		s.Suffix = "  Downloading "
		s.Start()

		reader, err := cli.ImagePull(ctx, "docker.io/library/gradle", types.ImagePullOptions{})
		if err != nil {
			panic(err)
		}
		io.Copy(ioutil.Discard, reader)
		s.Stop()
	}

	run := []string{
		headline("Downloading archive"),
		"cd /tmp",
		fmt.Sprintf("wget -v %s -O ./archive.zip", url),
		headline("Extracting archive"),
		"DIR=$(zipinfo -1 archive.zip | grep -oE '^[^/]+' | uniq)",
		"unzip archive.zip -d ./",
		"cd $DIR",
		headline("Compiling"),
		"mkdir ~/out",
		"gradle build",
		"mv build/libs/*.jar ~/out",
	}

	// create the docker container
	resp, err := cli.ContainerCreate(
		ctx,
		&container.Config{
			Image: "gradle",
			Cmd:   []string{"sh", "-c", strings.Join(run, ";")},
		},
		&container.HostConfig{
			Mounts: []mount.Mount{*getMount()},
		},
		nil, // no special networking config
		"",  // no special name
	)
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	// select {
	// case err := <-errCh:
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// case <-statusCh:
	// }

	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true, Follow: true})
	if err != nil {
		panic(err)
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)

	reader, _, err := cli.CopyFromContainer(ctx, resp.ID, "/home/gradle/out/")
	if err != nil {
		panic(err)
	}

	name, jar, err := extractJar(reader)
	if err != nil {
		logger.Fail(err.Error())
	}

	instance.Add(name, jar)
	cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{})
}

func extractJar(r io.Reader) (string, io.Reader, error) {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			panic(err)
		}

		filename := filepath.Base(hdr.Name)

		// not a jar file
		if !strings.HasSuffix(filename, ".jar") {
			continue
		}
		name := filename[:len(filename)-4] // filename without .jar
		// we don't want mod-dev.jar or mod-sources.jar only the blank one
		if strings.HasSuffix(name, "-dev") || strings.HasSuffix(name, "-sources") {
			continue
		}

		// should be our wanted jar file here
		return filename, tr, nil
	}
	return "", nil, errNoJarFile
}

// just a small helper to print something fancy
func headline(s string) string {
	s = aurora.BgGreen(" >> " + s + " ").String()
	return fmt.Sprintf(`echo "%s"`, s)
}

// get a mount dependent on the OS
func getMount() *mount.Mount {
	// Volume Binds are complicated on windows. They require additional setup from users
	if runtime.GOOS == "windows" {
		return &mount.Mount{
			Type:   mount.TypeVolume,
			Source: "minepkg-gradle",
			Target: "/home/gradle/.gradle",
		}
	}

	gradleCacheDir := filepath.Join(globalDir, "gradle")
	if err := os.MkdirAll(gradleCacheDir, 1755); err != nil {
		panic(err)
	}
	return &mount.Mount{
		Type:        mount.TypeBind,
		Source:      gradleCacheDir,
		Target:      "/home/gradle/.gradle",
		Consistency: mount.ConsistencyDelegated,
	}
}

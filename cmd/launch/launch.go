package launch

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"

	"github.com/charmbracelet/lipgloss"
	"github.com/jwalton/gchalk"
	"github.com/minepkg/minepkg/internals/commands"
	"github.com/minepkg/minepkg/internals/instances"
)

// Launch will launch the instance with the provided launchOptions
// and will set some fallback values
func (c *CLILauncher) Launch(opts *instances.LaunchOptions) error {
	fmt.Println("│")
	fmt.Println(
		lipgloss.JoinHorizontal(
			0.5,
			gchalk.Hex("#7a563b")("│"+"\n"+"┕"),
			commands.StyleGrass.Render(commands.Emoji("⛏  ")+"Launching Minecraft"),
		),
	)

	switch {
	case opts.LaunchManifest == nil:
		opts.LaunchManifest = c.LaunchManifest
	case !opts.Server:
		opts.Server = c.ServerMode
	}

	if c.UseSystemJava {
		opts.Java = "java"
	}

	cmd, err := c.Instance.BuildLaunchCmd(opts)
	if err != nil {
		return err
	}

	c.Cmd = cmd

	err = func() error {
		runtime.GC()
		if err := cmd.Start(); err != nil {
			return err
		}

		// we wait for the output to finish (the lines following this one usually are reached after ctrl-c was pressed)
		if err := cmd.Wait(); err != nil {
			return err
		}

		return nil
	}()

	// minecraft server will always return code 130 when
	// stop was successful, so we ignore the error here
	if cmd.ProcessState.ExitCode() == 130 || cmd.ProcessState.ExitCode() == 0 {
		fmt.Println("\nMinecraft was stopped normally")
		return nil
	}

	// TODO: what kind of errors are here?
	if err != nil {
		return err
	}

	if len(c.originalServerProps) != 0 {
		settingsFile := filepath.Join(c.Instance.McDir(), "server.properties")
		ioutil.WriteFile(settingsFile, c.originalServerProps, 0644)
	}

	return c.HandleCrash()
}

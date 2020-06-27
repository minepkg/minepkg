package launch

import (
	"github.com/fiws/minepkg/internals/instances"
)

// Launch will launch the instance with the provided launchOptions
// and will set some fallback values
func (c *CLILauncher) Launch(opts *instances.LaunchOptions) error {
	switch {
	case opts.LaunchManifest == nil:
		opts.LaunchManifest = c.LaunchManifest
	case opts.Server == false:
		opts.Server = c.ServerMode
	}

	cmd, err := c.Instance.BuildLaunchCmd(opts)
	if err != nil {
		return err
	}

	c.Cmd = cmd

	err = func() error {
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
	// stop was succesfull, so we ignore the error here
	if cmd.ProcessState.ExitCode() == 130 || cmd.ProcessState.ExitCode() == 0 {
		return nil
	}

	return c.HandleCrash()
}

package commands

import "github.com/spf13/cobra"

type Command struct {
	*cobra.Command
	runner Runner
}

type Runner interface {
	RunE(cmd *cobra.Command, args []string) error
}

func New(cmd *cobra.Command, run Runner) *Command {
	build := &Command{
		cmd,
		run,
	}
	build.Command.RunE = run.RunE

	return build
}

package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

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
	build.Command.Run = func(cmd *cobra.Command, args []string) {
		err := run.RunE(cmd, args)
		if err != nil {
			fmt.Println(err.Error() + "\n")
			os.Exit(1)
		}
	}

	return build
}

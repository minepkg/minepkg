package commands

import (
	"errors"
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
			var asCliErr *CliError
			if errors.As(err, &asCliErr) {
				fmt.Println(asCliErr.RichError() + "\n")
			} else {
				fmt.Println(
					ErrorBox(err.Error(), ""),
				)
			}
			os.Exit(1)
		}
	}

	return build
}

package cmd

import (
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/internals/commands"
	"github.com/spf13/cobra"
)

func init() {
	runner := &mpkgLoginRunner{}
	cmd := commands.New(&cobra.Command{
		Use:     "minepkg-login",
		Aliases: []string{"signin"},
		Short:   "Sign in to minepkg.io (mainly for publishing)",
		Args:    cobra.ExactArgs(0),
		Hidden:  true,
	}, runner)

	cmd.Flags().BoolVar(&runner.force, "force", false, "Always try to open the browser for login")

	rootCmd.AddCommand(cmd.Command)
}

type mpkgLoginRunner struct {
	force bool
}

func (i *mpkgLoginRunner) RunE(cmd *cobra.Command, args []string) error {

	if !i.force && !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		fmt.Println("This seems to be a server. You can not login on a server.")
		fmt.Println("Set the environment variable MINEPKG_API_KEY to a valid API key instead to authorize.")
		// TODO: create link
		fmt.Println("See https://minepkg.io/docs/ci for more info")
		fmt.Println("You can also add --force if you want to try to open a browser nonetheless")
		os.Exit(1)
	}

	fmt.Println("Trying to sign in to minepkg.io now …")
	fmt.Println("A browser window should open. Sign in there and click allow to continue.")

	oAuthConfig := api.OAuthLoginConfig{
		ClientID:     "minepkg-cli",
		ClientSecret: "",
		Scopes:       []string{"offline", "full_access"},
	}

	token := root.MinepkgAPI.OAuthLogin(&oAuthConfig)

	if err := root.setMinepkgAuth(token); err != nil {
		return err
	}
	fmt.Println(`Login to minepkg successful. You should now be able to publish packages!`)

	return nil
}

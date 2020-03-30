package cmd

import (
	"fmt"
	"os"

	"github.com/fiws/minepkg/pkg/api"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

func init() {
	mloginCmd.Flags().BoolVarP(&force, "force", "", false, "Try to open browser in any case to login")
	rootCmd.AddCommand(mloginCmd)
}

var mloginCmd = &cobra.Command{
	Use:     "minepkg-login",
	Aliases: []string{"signin"},
	Short:   "Sign in to minepkg.io",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		minepkgLogin()
	},
}

func minepkgLogin() {

	if !force && !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		fmt.Println("This seems to be a server. You can not login on a server.")
		fmt.Println("Set the environment variable MINEPKG_API_KEY to a valid API key instead to authorize.")
		// TODO: create link
		fmt.Println("See https://test-www.minepkg.io/docs/server-auth for more info")
		fmt.Println("You can also add --force if you want to try to open a browser nonetheless")
		os.Exit(1)
	}

	fmt.Println("Trying to sign to minepkg.io now …")
	fmt.Println("A browser window should open. Sign in there and click allow to continue.")

	oAuthConfig := api.OAuthLoginConfig{
		ClientID:     "minepkg-cli",
		ClientSecret: "",
		Scopes:       []string{"offline", "full_access"},
	}

	token := apiClient.OAuthLogin(&oAuthConfig)

	credStore.SetMinepkgAuth(token)
	fmt.Println(`Login to minepkg successful. You should now be able to publish packages!`)
}

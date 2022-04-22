package dev

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/internals/commands"
	"github.com/minepkg/minepkg/internals/globals"
	"github.com/minepkg/minepkg/internals/instances"
	"github.com/minepkg/minepkg/pkg/manifest"
	"github.com/spf13/cobra"
)

func init() {
	cmd := commands.New(&cobra.Command{
		Use:    "info [name/url/id]",
		Short:  "returns information on a single package",
		Hidden: false,
	}, &infoRunner{})

	cmd.Flags().String("minecraft", "*", "Overwrite the required Minecraft version")
	cmd.Flags().String("platform", "fabric", "Overwrite the wanted platform")
	cmd.Flags().Bool("lockfile", false, "Output lockfile instead of manifest")
	cmd.Flags().Bool("combined", false, "Output Combined manifest & lockfile")
	cmd.Flags().Bool("json", false, "Output json")

	SubCmd.AddCommand(cmd.Command)
}

type infoRunner struct{}

func (i *infoRunner) RunE(cmd *cobra.Command, args []string) error {
	apiClient := globals.ApiClient

	if len(args) == 0 {
		instance, err := instances.NewFromWd()
		if err != nil {
			return err
		}

		wantsJson, _ := cmd.Flags().GetBool("json")
		wantsCombined, _ := cmd.Flags().GetBool("combined")
		wantsLockfile, _ := cmd.Flags().GetBool("lockfile")

		if wantsJson {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.SetEscapeHTML(false)

			var toEncode interface{}
			switch {
			case wantsCombined:
				toEncode = &struct {
					Manifest *manifest.Manifest `json:"manifest"`
					Lockfile *manifest.Lockfile `json:"lockfile"`
				}{
					Manifest: instance.Manifest,
					Lockfile: instance.Lockfile,
				}
			case wantsLockfile:
				toEncode = instance.Lockfile
			default:
				toEncode = instance.Manifest
			}

			err := enc.Encode(toEncode)
			return err
		}
		fmt.Println(instance.Manifest)
		return nil
	}

	comp := strings.Split(args[0], "@")
	name := comp[0]
	version := "latest"
	reqsMinecraft, _ := cmd.Flags().GetString("minecraft")
	platform, _ := cmd.Flags().GetString("platform")
	if len(comp) == 2 {
		version = comp[1]
	}

	fmt.Println("Searching for:")
	fmt.Printf(
		"  provider: %s\n  name: %s\n  version: %s\n  reqs.minecraft: %s\n",
		"minepkg",
		name,
		version,
		reqsMinecraft,
	)

	r, err := apiClient.FindRelease(context.TODO(), name, &api.RequirementQuery{
		Minecraft: reqsMinecraft,
		Platform:  platform,
		Version:   version,
	})

	if err != nil {
		return err
	}

	fmt.Println("\nFound package manifest:")
	fmt.Println(r)

	fmt.Println("tested working with:")
	for _, test := range r.Tests {
		if test.Works {
			fmt.Printf(" %s ", test.Minecraft)
		}
	}
	fmt.Println()
	return nil
}

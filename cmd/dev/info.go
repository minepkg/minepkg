package dev

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/internals/commands"
	"github.com/minepkg/minepkg/internals/globals"
	"github.com/minepkg/minepkg/internals/instances"
	"github.com/minepkg/minepkg/internals/utils"
	"github.com/minepkg/minepkg/pkg/manifest"
	"github.com/spf13/cobra"
	"golang.org/x/exp/constraints"
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
		"  name: %s\n  version: %s\n  reqs.minecraft: %s\n",
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

	stats, err := apiClient.GetProjectStats(context.TODO(), name)
	if err != nil {
		return err
	}

	fmt.Println("\nFound package on minepkg:")

	twoWeekDownloads := make([]int, 14)
	dateIndex := time.Now().Add(-time.Hour * 24 * 14)

	// stats.Downloads is sorted by date, but is missing days with 0 downloads
	// so we need to fill in the gaps
	for i, d := range stats.Downloads {
		for dateIndex.Before(d.Date) {
			twoWeekDownloads[i] = 0
			dateIndex = dateIndex.Add(time.Hour * 24)
			i++
		}
		twoWeekDownloads[i] = d.Downloads
		dateIndex = dateIndex.Add(time.Hour * 24)
	}

	bold := lipgloss.NewStyle().MaxWidth(11).Bold(true)

	row := func(key, value string) string {
		return lipgloss.JoinHorizontal(lipgloss.Left,
			bold.PaddingLeft(11-len(key)).Render(key),
			lipgloss.NewStyle().Width(60).Padding(0, 1).Render(value),
		)
	}

	box := lipgloss.Style{}.MaxWidth(80).Padding(1, 2).
		Render(
			strings.Join([]string{
				row("Name", r.Package.Name),
				row("Description", r.Package.Description),
				row("Version", r.Package.Version),
				row("Minecraft", r.Requirements.Minecraft),
				row("Platform", r.Package.Platform),
				row("Author", r.Package.Author),
				row("License", r.Package.License),
				row(
					"Downloads",
					utils.HumanInteger(stats.Summary.TotalDownloads)+" "+
						lipgloss.NewStyle().Foreground(lipgloss.Color("#3399aa")).Render(renderSparkLine(twoWeekDownloads)),
				),
			}, "\n"),
		)

	fmt.Println(box)

	fmt.Println("\nConfirmed working:")
	for _, test := range r.Tests {
		if test.Works {
			fmt.Printf(" %s ", test.Minecraft)
		}
	}
	fmt.Println()
	return nil
}

var bars = []string{
	" ", "⢀", "⢠", "⢰", "⢸",
	"⡀", "⣀", "⣠", "⣰", "⣸",
	"⡄", "⣄", "⣤", "⣴", "⣼",
	"⡆", "⣆", "⣦", "⣶", "⣾",
	"⡇", "⣇", "⣧", "⣷", "⣿",
}

func renderSparkLine[T constraints.Integer | constraints.Float](values []T) string {
	min := values[0]
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
		if v < min {
			min = v
		}
	}

	normalized := make([]int, len(values))
	for i, v := range values {
		// normalize to 0-4
		normalized[i] = int(float64(v-min) / float64(max-min) * 4)
		if normalized[i] == 0 && v != 0 {
			normalized[i] = 1
		}
	}

	// render in pairs of 2
	downloadsBar := make([]string, len(normalized))
	for i := 0; i < len(normalized); i += 2 {
		a := normalized[i]
		b := 0
		if i+1 < len(normalized) {
			b = normalized[i+1]
		}
		downloadsBar[i] = bars[a*5+b]
	}

	return strings.Join(downloadsBar, "")
}

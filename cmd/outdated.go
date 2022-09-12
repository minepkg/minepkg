package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/Masterminds/semver/v3"
	"github.com/charmbracelet/lipgloss"
	"github.com/jwalton/gchalk"
	"github.com/minepkg/minepkg/internals/instances"
	"github.com/minepkg/minepkg/internals/provider"
	"github.com/minepkg/minepkg/internals/utils"
	"github.com/minepkg/minepkg/pkg/manifest"
	"github.com/spf13/cobra"
)

var (
	MinecraftVersionFlag string
)

// outdatedCmd represents the outdated command
var outdatedCmd = &cobra.Command{
	Use:    "outdated",
	Short:  "Returns a list of outdated dependencies",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		instance, err := root.LocalInstance()
		if err != nil {
			return err
		}

		minecraftReq := MinecraftVersionFlag
		if MinecraftVersionFlag == "" {
			minecraftReq = instance.Lockfile.MinecraftVersion()
		}

		fmt.Printf("Checking %s for outdated packages\n", instance.Manifest.Package.Name)
		fmt.Printf("Minecraft %s\n", minecraftReq)
		fmt.Println("  Hint: Use --minecraft to check against a different version")
		fmt.Print("  Example: minepkg outdated --minecraft 1.18.1\n\n")

		table := table{}
		table.addColumn("Package", 30)
		table.addColumn("Provider", 10)
		table.addColumn("Current", 23)
		table.addColumn("Latest (for "+minecraftReq+")", 40)

		numOk := 0

		dependencies := instance.GetDependencyList().Sorted()
		for _, dependency := range dependencies {
			if dependency.ID.Version == "none" {
				continue
			}
			platformLock := instance.Lockfile.PlatformLock()
			latestRequest := instances.ProviderRequest(&dependency, platformLock)
			var latest provider.Result
			rowStyle := lipgloss.NewStyle()
			latestVersion := "unknown"
			if latestRequest.Dependency.Provider != "dummy" {
				customRequirement := &manifest.PlatformRequirement{
					Minecraft:     minecraftReq,
					LoaderName:    platformLock.PlatformName(),
					LoaderVersion: platformLock.PlatformVersion(),
				}

				latestRequest.Requirements = customRequirement
				latest, err = root.ProviderStore.ResolveLatest(context.TODO(), latestRequest)
				if err != nil {
					log.Println(err)
					latestVersion = "unavailable"
					rowStyle = lipgloss.NewStyle().Faint(true)
					// TODO: handle error
				}
			}

			if latest != nil {
				if dependency.Lock == nil {
					return fmt.Errorf("dependency %s has no lock. please run minepkg install", dependency.Name())
				}
				// already latest version?
				if latest.Lock().Version == dependency.Lock.Version {
					numOk++

					log.Println(gchalk.Green("✓"), dependency.Name(), "is up to date")

					continue
				}
				latestVersion = latest.Lock().Version

				if latest.Lock().VersionName != "" {
					latestVersion = fmt.Sprintf("%s (%s)", latest.Lock().VersionName, latest.Lock().Version)
				}

				if latest.Lock().Provider == "minepkg" {
					oldSemver := semver.MustParse(dependency.Lock.Version)
					newSemver := semver.MustParse(latest.Lock().Version)

					latestVersion = utils.PrettyVersion(latest.Lock().Version)

					switch {
					case oldSemver.Major() != newSemver.Major():
						rowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FC4F4F"))
					case oldSemver.Minor() != newSemver.Minor():
						rowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F76E11"))
					case oldSemver.Patch() != newSemver.Patch():
						rowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF9F45"))
					default:
						rowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFBC80"))
					}
				} else {
					rowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFBC80"))
				}
			}

			var currentVersion string
			if dependency.Lock.VersionName != "" {
				currentVersion = fmt.Sprintf("%s (%s)", dependency.Lock.VersionName, dependency.Lock.Version)
			} else {
				currentVersion = utils.PrettyVersion(dependency.Lock.Version)
			}

			row := table.addRow([]string{
				dependency.Name(),
				dependency.ID.Provider,
				currentVersion,
				latestVersion,
			})

			row.Style = rowStyle
		}

		fmt.Println(table.render())

		if numOk > 0 {
			note := fmt.Sprintf("%d dependencies hidden that where up to date.\n", numOk)
			fmt.Println(gchalk.BrightGreen(note))
		}

		return nil
	},
}

type tableColumn struct {
	Width       int
	Name        string
	Style       lipgloss.Style
	HeaderStyle lipgloss.Style
}

type tableRow struct {
	Cells []string
	Style lipgloss.Style
}

type table struct {
	columns []tableColumn
	rows    []*tableRow
}

func (t *table) addColumn(name string, width int) *table {
	t.columns = append(t.columns, tableColumn{
		Width:       width,
		Name:        name,
		Style:       lipgloss.NewStyle().Width(width).PaddingRight(1),
		HeaderStyle: lipgloss.NewStyle().Width(width).Bold(true).Underline(true).PaddingRight(1),
	})

	return t
}

func (t *table) addRow(data []string) *tableRow {
	newRow := &tableRow{
		Cells: data,
		Style: lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")),
	}
	t.rows = append(t.rows, newRow)
	return newRow
}

func (t *table) render() string {
	var rendered string
	for _, column := range t.columns {
		rendered += lipgloss.JoinHorizontal(lipgloss.Left, column.HeaderStyle.Render(column.Name))
	}
	rendered += "\n"
	for _, row := range t.rows {
		renderedCells := make([]string, len(row.Cells))
		for i, column := range t.columns {
			renderedCells[i] = column.Style.Render(row.Cells[i])
		}
		rendered += row.Style.Render(lipgloss.JoinHorizontal(
			lipgloss.Left,
			renderedCells...,
		)) + "\n"
	}
	return rendered
}

func init() {
	rootCmd.AddCommand(outdatedCmd)

	outdatedCmd.Flags().StringVar(&MinecraftVersionFlag, "minecraft", "", "Minecraft version to check against")
}

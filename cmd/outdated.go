package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/charmbracelet/lipgloss"
	"github.com/jwalton/gchalk"
	"github.com/minepkg/minepkg/internals/provider"
	"github.com/minepkg/minepkg/internals/utils"
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

		// start timer
		start := time.Now()

		fmt.Println("minimum", time.Since(start))

		result, err := instance.Outdated(context.Background())
		if err != nil {
			return err
		}

		sort.Slice(result, func(i, j int) bool {
			return result[i].Dependency.Name() < result[j].Dependency.Name()
		})

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

		for _, entry := range result {
			if entry.Dependency.ID.Version == "none" {
				continue
			}

			currentLock := entry.Dependency.Lock

			var rowStyle lipgloss.Style
			latestVersion := "unknown"

			if entry.Error != nil {
				if errors.Is(entry.Error, provider.ErrProviderUnsupported) {
					continue
				}
				log.Println(err)
				latestVersion = "unavailable (" + entry.Error.Error() + ")"
				rowStyle = lipgloss.NewStyle().Faint(true)
				// TODO: handle error
				// if dependency.Lock == nil {
				// 	return fmt.Errorf("dependency %s has no lock. please run minepkg install", dependency.Name())
				// }
			} else {
				// already latest version?
				if entry.Result.Lock().Version == entry.Dependency.Lock.Version {
					numOk++

					log.Println(gchalk.Green("✓"), entry.Dependency.Name(), "is up to date")

					continue
				}

				latestLock := entry.Result.Lock()
				latestVersion = latestLock.Version

				if latestVersion != "" {
					latestVersion = fmt.Sprintf("%s (%s)", latestLock.VersionName, latestLock.Version)
				}

				if entry.Result.Lock().Provider == "minepkg" {
					oldSemver := semver.MustParse(entry.Dependency.Lock.Version)
					newSemver := semver.MustParse(latestLock.Version)

					latestVersion = utils.PrettyVersion(latestLock.Version)

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
			if currentLock.VersionName != "" {
				currentVersion = fmt.Sprintf("%s (%s)", currentLock.VersionName, currentLock.Version)
			} else {
				currentVersion = utils.PrettyVersion(currentLock.Version)
			}

			row := table.addRow([]string{
				entry.Dependency.Name(),
				entry.Dependency.ID.Provider,
				currentVersion,
				latestVersion,
			})

			row.Style = rowStyle
		}

		fmt.Println(table.render())

		if numOk > 0 {
			note := fmt.Sprintf("%d dependencies hidden that are up to date.\n", numOk)
			fmt.Println(gchalk.BrightGreen(note))
		}

		fmt.Println("Took", time.Since(start))

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

package autocomplete

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/internals/utils"
	"github.com/spf13/cobra"
)

type AutoCompleter struct {
	Client  *api.MinepkgClient
	storage struct {
		LastFetch            time.Time
		Projects             []api.Project
		MaxDownloadsMods     uint32
		MaxDownloadsModpacks uint32
	}
	CacheDir string
}

func (a *AutoCompleter) cacheFile() string {
	return path.Join(a.CacheDir, "packages.json")
}

func (a *AutoCompleter) isOutdated() bool {
	return time.Since(a.storage.LastFetch) > 10*time.Second
}

// GetPackages tries to fetch the packages from the local cache
// if that fails it will fetch them from the api
func (a *AutoCompleter) GetProjects(ctx context.Context) ([]api.Project, error) {
	// already in memory and not older than 10 seconds
	if a.storage.Projects != nil && !a.isOutdated() {
		return a.storage.Projects, nil
	}

	// read the local cache file
	// try to read the file
	pkgs, err := os.ReadFile(a.cacheFile())
	if err != nil {
		// if the file doesn't exist, fetch the packages from the api
		return a.fetchPackages(ctx)
	}

	// if the file exists, unmarshal it into a.packages
	err = json.Unmarshal(pkgs, &a.storage)
	if err != nil {
		// if the file is corrupted, fetch the packages from the api
		return a.fetchPackages(ctx)
	}

	// we update the cache if it is outdated
	if a.isOutdated() {
		projects, err := a.fetchPackages(ctx)
		if err == nil {
			return projects, nil
		}
		// if the api is down, we still want to return the cached packages (if they exist)
		// so we just ignore the error
	}

	// if the file exists and is not corrupted, return the packages
	return a.storage.Projects, nil
}

func (a *AutoCompleter) fetchPackages(ctx context.Context) ([]api.Project, error) {
	// fetch all packages from the api
	// and store them in a.packages
	projectsQuery := api.GetProjectsQuery{Simple: true}
	projects, err := a.Client.GetProjects(ctx, &projectsQuery)
	if err != nil {
		return nil, err
	}

	// sort the packages by downloads
	sortByDownloads(projects)

	// find the max downloads for mods and modpacks
	var maxDownloadsMods, maxDownloadsModpacks uint32
	for _, p := range projects {
		if len(p.Categories) == 1 && p.Categories[0] == "library" {
			continue
		}
		if p.Type == "mod" && maxDownloadsMods == 0 {
			maxDownloadsMods = p.Stats.TotalDownloads
		}
		if p.Type == "modpack" && maxDownloadsModpacks == 0 {
			maxDownloadsModpacks = p.Stats.TotalDownloads
			break
		}
	}

	a.storage.Projects = projects
	a.storage.LastFetch = time.Now()
	a.storage.MaxDownloadsModpacks = maxDownloadsModpacks
	a.storage.MaxDownloadsMods = maxDownloadsMods
	// write the packages to the cache file
	pkgs, err := json.Marshal(&a.storage)
	if err != nil {
		return projects, err
	}
	err = os.WriteFile(a.cacheFile(), pkgs, 0644)
	return projects, err
}

func (a *AutoCompleter) Complete(toComplete string) ([]string, cobra.ShellCompDirective) {
	// error is ignored on purpose
	projects, _ := a.GetProjects(context.TODO())
	if projects == nil {
		// we can't error here, so just return an empty list
		projects = []api.Project{}
	}

	return a.shellAutocomplete(projects, toComplete)
}

func (a *AutoCompleter) CompleteModpacks(toComplete string) ([]string, cobra.ShellCompDirective) {
	// error is ignored on purpose
	projects, _ := a.GetProjects(context.TODO())
	if projects == nil {
		// we can't error here, so just return an empty list
		projects = []api.Project{}
	}

	// filter out all non modpacks
	var modpacks []api.Project
	for _, p := range projects {
		if p.Type == "modpack" {
			modpacks = append(modpacks, p)
		}
	}

	return a.shellAutocomplete(modpacks, toComplete)
}

func (a *AutoCompleter) shellAutocomplete(projects []api.Project, toComplete string) ([]string, cobra.ShellCompDirective) {
	// find all packages that start with toComplete
	var matches []string
	for _, p := range projects {
		if strings.HasPrefix(p.Name, toComplete) {
			maxDl := a.storage.MaxDownloadsMods
			icon := ""
			if p.Type == "modpack" {
				maxDl = a.storage.MaxDownloadsModpacks
				icon = "📦 "
			}
			popularityIndicator := renderPopularity(p.Stats.TotalDownloads, maxDl)
			dlIndicator := lipgloss.NewStyle().Width(2).Render(popularityIndicator)
			count := lipgloss.NewStyle().Width(4).Align(lipgloss.Right).Render(utils.HumanInteger(p.Stats.TotalDownloads))

			description := fmt.Sprintf(
				"%s %s | %s%s",
				dlIndicator,
				count,
				icon,
				p.Description,
			)
			line := fmt.Sprintf("%s\t%s", p.Name, description)
			matches = append(matches, line)
		}
	}
	return matches, cobra.ShellCompDirectiveNoFileComp
}

func renderPopularity(downloadCount uint32, maxDownloads uint32) string {
	bars := []string{" ", "🥉", "🥈", "🥇"}
	// calculate the percentage of downloads
	percentage := 0.0
	if downloadCount > 0 && maxDownloads > 0 {
		percentage = math.Log(float64(downloadCount)) / math.Log(float64(maxDownloads))
	}
	if percentage > 1 {
		percentage = 1
	}

	// calculate the bar index
	barIndex := int(percentage * float64(len(bars)-1))
	// return the correct bar
	return bars[barIndex]
}

func sortByDownloads(projects []api.Project) {
	// sort the packages by downloads
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Stats.TotalDownloads > projects[j].Stats.TotalDownloads
	})
}

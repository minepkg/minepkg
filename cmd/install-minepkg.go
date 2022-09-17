package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/erikgeiser/promptkit/confirmation"
	"github.com/jwalton/gchalk"
	"github.com/manifoldco/promptui"
	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/internals/commands"
	"github.com/minepkg/minepkg/internals/downloadmgr"
	"github.com/minepkg/minepkg/internals/globals"
)

func (i *installRunner) installFromMinepkg(mods []string) error {
	instance := i.instance
	apiClient := globals.ApiClient

	task := logger.NewTask(3)
	task.Step("📚", "Finding packages")

	releases := make([]*api.Release, len(mods))

	s := spinner.New(spinner.CharSets[9], 300*time.Millisecond) // Build our new spinner
	s.Prefix = " "

	mgr := downloadmgr.New()
	mgr.OnProgress = func(p int) {
		s.Suffix = fmt.Sprintf(" Downloading %v", p) + "%"
	}

	// resolve requirements
	if instance.Lockfile == nil || !instance.Lockfile.HasRequirements() {
		s.Suffix = " Resolving Requirements"
		instance.UpdateLockfileRequirements(context.TODO())
		instance.SaveLockfile()
	}

	for i, name := range mods {
		comp := strings.Split(name, "@")
		name = comp[0]
		version := "latest"
		if len(comp) == 2 {
			version = comp[1]
		}

		reqs := &api.ReleasesQuery{
			Name:         name,
			VersionRange: version,
			Minecraft:    instance.Lockfile.MinecraftVersion(),
			Platform:     instance.Manifest.PlatformString(),
		}

		release, err := apiClient.ReleasesQuery(context.TODO(), reqs)
		if err != nil {

			// package names have to be exact for multi-package installs
			// we skip the fallback search here
			if len(mods) >= 2 {
				return err
			}

			// TODO: check if this was a 404
			mod := searchFallback(context.TODO(), name)
			if mod == nil {
				return err
			}
			newQuery := &api.ReleasesQuery{
				Name:         mod.Name,
				VersionRange: version,
				Minecraft:    reqs.Minecraft,
				Platform:     reqs.Platform,
			}
			release, err = apiClient.ReleasesQuery(context.TODO(), newQuery)
			if err != nil {
				return prettyApiError(err)
			}
		}
		if release == nil {
			logger.Info("Could not find package " + name + "@" + version)
			os.Exit(1)
		}
		releases[i] = release
	}

	if len(releases) == 1 {
		logger.Info("Installing " + releases[0].Package.Name + "@" + releases[0].Package.Version)
	} else {
		input := confirmation.New(fmt.Sprintf("Install %d mods?", len(releases)), confirmation.Yes)

		choice, err := input.RunPrompt()
		if err != nil || choice == false {
			logger.Info("Aborting installation")
			os.Exit(0)
		}
	}

	task.Step("🔎", "Resolving Dependencies")
	for _, release := range releases {
		if !i.dev {
			instance.Manifest.AddDependency(release.Package.Name, "^"+release.Package.Version)
		} else {
			fmt.Println("Adding as dev dependency!")
			instance.Manifest.AddDevDependency(release.Package.Name, "^"+release.Package.Version)
		}
	}

	instance.UpdateLockfileDependencies(context.TODO())
	for _, dep := range instance.Lockfile.Dependencies {
		fmt.Printf(" - %s@%s\n", dep.Name, dep.Version)
	}
	missingFiles, err := instance.FindMissingDependencies()
	if err != nil {
		logger.Fail(err.Error())
	}

	task.Step("🚚", fmt.Sprintf("Downloading %d Packages", len(missingFiles)))
	for _, m := range missingFiles {
		p := filepath.Join(instance.PackageCacheDir(), m.Name, m.Version+m.FileExt())
		item := downloadmgr.HTTPItem{URL: m.URL, Target: p, Sha256: m.Sha256}
		mgr.Add(&item)
	}

	s.Start()
	if err := mgr.Start(context.TODO()); err != nil {
		logger.Fail(err.Error())
	}

	instance.LinkDependencies()

	s.Stop()
	instance.SaveManifest()
	instance.SaveLockfile()
	fmt.Println("updated minepkg.toml")
	fmt.Println("You can now launch Minecraft using \"minepkg launch\"")

	return nil
}

func searchFallback(ctx context.Context, name string) *api.Project {
	projects, _ := globals.ApiClient.GetProjects(ctx, &api.GetProjectsQuery{})

	filtered := make([]api.Project, 0, 10)
	for _, p := range projects {
		if strings.Contains(p.Name, name) {
			filtered = append(filtered, p)
		}
	}

	if len(filtered) == 0 {
		return nil
	}

	if len(filtered) == 1 {
		prompt := promptui.Prompt{
			Label:     fmt.Sprintf("Autocomplete to %s", filtered[0].Name),
			IsConfirm: true,
			Default:   "Y",
		}

		_, err := prompt.Run()
		if err != nil {
			logger.Info("Aborting installation")
			os.Exit(0)
		}
		return &filtered[0]
	}

	fmt.Println("Found multiple packages by that name, please select one.")

	selectable := make([]string, len(filtered))
	for i, mod := range filtered {
		selectable[i] = fmt.Sprintf("%s (%v Downloads)", mod.Name, HumanUint32(mod.Stats.TotalDownloads))
	}

	prompt := promptui.Select{
		Label: "Select Package",
		Items: selectable,
		Size:  8,
	}

	i, _, err := prompt.Run()
	if err != nil {
		fmt.Println("Aborting installation")
		os.Exit(0)
	}
	return &filtered[i]
}

func prettyApiError(err error) error {
	var notFoundErr *api.ErrNoMatchingRelease
	if errors.As(err, &notFoundErr) {
		switch notFoundErr.Err {
		case api.ErrProjectDoesNotExist:
			return &commands.CliError{
				Text: fmt.Sprintf("Project %s does not exist", notFoundErr.Package),
				Suggestions: []string{
					"Check if your have a typo in the package name",
					"Make sure the wanted Project is published",
				},
			}
		case api.ErrNoReleasesForPlatform:
			return &commands.CliError{
				Text: fmt.Sprintf(
					"Project %s has no releases for %s",
					notFoundErr.Package,
					notFoundErr.Requirements.Platform,
				),
			}
		case api.ErrNoReleaseForMinecraftVersion:
			return &commands.CliError{
				Text: fmt.Sprintf(
					"Project %s is not compatible with Minecraft %s",
					notFoundErr.Package,
					notFoundErr.Requirements.Minecraft,
				),
				Suggestions: []string{
					fmt.Sprintf("Change the %s field in your minepkg.toml", gchalk.Bold("requirements.minecraft")),
					"Wait until the Author publishes a release for this version",
				},
			}
		case api.ErrNoReleaseForVersion:
			return &commands.CliError{
				Text: fmt.Sprintf(
					"Project %s with version requirement %s not found",
					notFoundErr.Package,
					notFoundErr.Requirements.Version,
				),
				Suggestions: []string{
					fmt.Sprintf("Install a different version of this package (%s)", gchalk.Bold("minepkg install "+notFoundErr.Package+"@version")),
				},
			}
		case api.ErrNoReleaseForVersion:
			return &commands.CliError{
				Text: fmt.Sprintf(
					"Project %s not compatible with current requirements",
					notFoundErr.Package,
				),
				Suggestions: []string{
					fmt.Sprintf("Install a different version of this package (%s)", gchalk.Bold("minepkg install "+notFoundErr.Package+"@version")),
					fmt.Sprintf("Change the %s field in your minepkg.toml", gchalk.Bold("requirements.minecraft")),
				},
			}
		}
	}

	return err
}

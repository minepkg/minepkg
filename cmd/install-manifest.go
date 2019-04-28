package cmd

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fiws/minepkg/internals/curse"
	"github.com/fiws/minepkg/internals/instances"
)

// installManifest installs dependencies from the minepkg.toml
func installManifest(instance *instances.McInstance) {
	task := logger.NewTask(3)

	task.Step("📚", "Searching local mod DB.")
	db := readDbOrDownload()

	deps := &instance.Manifest.Dependencies
	// if err != nil {
	// 	task.Fail("Failed to extend " + err.Error())
	// }

	mods := make([]*curse.Mod, len(*deps))

	i := 0
	for name := range *deps {
		resolved := db.modBySlug(name)
		if resolved == nil {
			task.Fail("Could not resolve " + name)
		}
		mods[i] = resolved
		i++
	}

	task.Step("🔎", "Resolving Dependencies")
	resolver := curse.NewResolver()

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond) // Build our new spinner
	s.Prefix = "  "
	s.Start()
	for _, mod := range mods {
		s.Suffix = "  Resolving " + mod.Slug
		resolver.Resolve(mod.ID, instance.Manifest.Requirements.Minecraft)
	}
	s.Stop()
	resolved := resolver.Resolved

	for _, mod := range resolved {
		task.Log(fmt.Sprintf("requires %s", mod.FileName))
	}

	task.Step("🚚", "Downloading Mods")

	for _, mod := range resolved {
		err := instance.Download(mod.FileName, mod.DownloadURL)
		if err != nil {
			logger.Fail(fmt.Sprintf("Could not download %s (%s)"+mod.FileName, err))
		}
		task.Log(fmt.Sprintf("downloaded %s", mod.FileName))
	}
}

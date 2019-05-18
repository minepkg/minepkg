package cmd

import (
	"fmt"

	"github.com/fiws/minepkg/internals/instances"
	"github.com/fiws/minepkg/pkg/api"
)

// installManifest installs dependencies from the minepkg.toml
func installManifest(instance *instances.McInstance) {

	task := logger.NewTask(2)

	task.Info("Installing minepkg.toml dependencies")

	task.Step("🔎", "Resolving Dependencies")
	res := api.NewResolver(apiClient)
	res.ResolveManifest(instance.Manifest)

	// logger.Info("The following Dependencies will be downloaded:")
	// logger.Info(strings.Join())
	task.Step("🚚", "Downloading Packages")

	for _, p := range res.Resolved {
		task.Log("Downloading " + p.Project + "@" + p.Version)
		err := instance.Download(p.Project+".jar", p.DownloadURL())
		if err != nil {
			logger.Fail(fmt.Sprintf("Could not download %s (%s)"+p.Project, err))
		}
	}
}

package cmd

import (
	"context"
	"fmt"
	"github.com/fiws/minepkg/pkg/manifest"
	"os"
	"strings"

	"github.com/fiws/minepkg/pkg/api"

	"github.com/fiws/minepkg/internals/instances"
	"github.com/manifoldco/promptui"
)

// Resolver resolves given the mods of given dependencies
type Resolver struct {
	Resolved map[string]*api.Release
}

// NewResolver returns a new resolver
func NewResolver() *Resolver {
	return &Resolver{Resolved: make(map[string]*api.Release)}
}

// ResolveManifest resolves a manifest
func (r *Resolver) ResolveManifest(man *manifest.Manifest) error {

	for name, version := range man.Dependencies {
		release, err := apiClient.FindRelease(context.TODO(), name, version)
		if err != nil {
			return err
		}
		r.ResolveSingle(release)
	}

	return nil
}

// Resolve find all dependencies from the given `id`
// and adds it to the `resolved` map. Nothing is returned
func (r *Resolver) Resolve(releases []*api.Release) error {
	for _, release := range releases {
		r.ResolveSingle(release)
	}

	return nil
}

// ResolveSingle resolves all dependencies of a single release
func (r *Resolver) ResolveSingle(release *api.Release) error {

	r.Resolved[release.Project] = release
	// TODO: parallelize
	for _, d := range release.Dependencies {
		_, ok := r.Resolved[d.Name]
		if ok == true {
			return nil
		}
		r.Resolved[d.Name] = nil
		release, err := d.Resolve(context.TODO())
		if err != nil {
			return err
		}
		r.ResolveSingle(release)
	}

	return nil
}

func installFromMinepkg(name string, instance *instances.McInstance) {

	task := logger.NewTask(3)
	task.Step("📚", "Searching requested package")
	// db := readDbOrDownload()

	// // TODO: better search!
	// mods := curse.Filter(db.Mods, func(m curse.Mod) bool {
	// 	return strings.HasPrefix(strings.ToLower(m.Slug), name)
	// })

	// choosenMod := chooseMod(mods, task)

	comp := strings.Split(name, "@")
	name = comp[0]
	version := "latest"
	if len(comp) == 2 {
		version = comp[1]
	}
	release, _ := apiClient.FindRelease(context.TODO(), name, version)

	if release == nil {
		logger.Info("Could not find package " + name + "@" + version)
		os.Exit(1)
	}

	prompt := promptui.Prompt{
		Label:     "Install " + name + "@" + release.Version,
		IsConfirm: true,
		Default:   "Y",
	}

	_, err := prompt.Run()
	if err != nil {
		logger.Info("Aborting installation")
		os.Exit(0)
	}

	task.Step("🔎", "Resolving Dependencies")
	res := NewResolver()
	res.ResolveSingle(release)
	instance.Manifest.AddDependency(release.Project, release.Version)

	// logger.Info("The following Dependencies will be downloaded:")
	// logger.Info(strings.Join())
	task.Step("🚚", "Downloading Packages")

	for _, p := range res.Resolved {
		task.Log("Downloading " + p.Project + "@" + p.Version)
		err = instance.Download(p.Project+".jar", p.DownloadURL())
		if err != nil {
			logger.Fail(fmt.Sprintf("Could not download %s (%s)"+p.Project, err))
		}
	}
	instance.Manifest.Save()
	fmt.Println("updated minepkg.toml")

}

// func chooseMod(mods []curse.Mod, task *cmdlog.Task) *curse.Mod {
// 	modCount := len(mods)
// 	var choosen *curse.Mod
// 	switch {
// 	case modCount == 0:
// 		task.Fail("Found no matching packages by that name")
// 	case modCount == 1:
// 		choosen = &mods[0]
// 		prompt := promptui.Prompt{
// 			Label:     "Install " + choosen.Name,
// 			IsConfirm: true,
// 			Default:   "Y",
// 		}

// 		_, err := prompt.Run()
// 		if err != nil {
// 			logger.Info("Aborting installation")
// 			os.Exit(0)
// 		}
// 	default:
// 		task.Info("Found multiple packages by that name, please select one.")
// 		curse.SortByDownloadCount(mods)

// 		selectable := make([]string, modCount)
// 		for i, mod := range mods {
// 			selectable[i] = fmt.Sprintf("%s (%v)", mod.Name, HumanFloat32(mod.DownloadCount))
// 		}

// 		prompt := promptui.Select{
// 			Label: "Select Package",
// 			Items: selectable,
// 			Size:  8,
// 		}

// 		i, _, err := prompt.Run()
// 		if err != nil {
// 			fmt.Println("Aborting installation")
// 			os.Exit(0)
// 		}
// 		choosen = &mods[i]
// 	}
// 	return choosen
// }

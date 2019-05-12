package instances

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	homedir "github.com/mitchellh/go-homedir"
)

var (
	// ErrorLaunchNotImplemented is returned if attemting to start a non vanilla instance
	ErrorLaunchNotImplemented = errors.New("Can only launch vanilla instances (for now)")
	// ErrorNoCredentials is returned when an instance is launched without `MojangProfile` beeing set
	ErrorNoCredentials = errors.New("Can not launch without mojang credentials")
	// ErrorNoFabricLoader is returned if the wanted fabric version was not found
	ErrorNoFabricLoader = errors.New("Could not find wanted fabric version")
)

// Launch starts the minecraft instance
func (m *McInstance) Launch() error {
	home, _ := homedir.Dir()
	globalDir := filepath.Join(home, ".minepkg")
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	creds := m.MojangCredentials
	profile := creds.SelectedProfile
	if profile == nil {
		return ErrorNoCredentials
	}

	// instructions might be a bad name. this file tells us howto construct the start command
	instr, err := m.launchManifest()
	if err != nil {
		return err
	}

	if instr.InheritsFrom != "" {
		parent, err := m.getLaunchManifest(instr.InheritsFrom)
		if err != nil {
			return err
		}
		instr.MergeWith(parent)
	}

	m.ensureAssets(instr)

	tmpName := m.Manifest.Package.Name + fmt.Sprintf("%d", time.Now().Unix())
	tmpDir, err := ioutil.TempDir("", tmpName)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir) // cleanup dir
	libDir := filepath.Join(globalDir, "libraries")

	// build that spooky -cp arg
	// TODO: use rules! some libs have to be excluded on osx
	var cpArgs []string

	for _, lib := range instr.Libraries {

		skip := false
		for _, rule := range lib.Rules {
			switch {
			// allow block but does not match os
			case rule.Action == "allow" && rule.OS.Name != runtime.GOOS:
				skip = true
			// disallow block matches os
			case rule.Action == "disallow" && rule.OS.Name == runtime.GOOS:
				skip = true
			// don't skip otherwise
			default:
				skip = false
			}
		}
		// skip platform files
		if skip == true {
			continue
		}

		// copy natives. not sure if this implementation is complete
		if len(lib.Natives) != 0 {
			nativeID, ok := lib.Natives[runtime.GOOS]
			// skip native not available for this platform
			if ok != true {
				continue
			}
			// extract native to temp dir
			native := lib.Downloads.Classifiers[nativeID]

			p := filepath.Join(libDir, native.Path)
			existOrDownload(lib)
			err := extractNative(p, tmpDir)
			if err != nil {
				return err
			}
			cpArgs = append(cpArgs, filepath.Join(libDir, native.Path))
			continue
		}
		// not skipped. append this library to our doom -cp arg
		libPath := lib.Downloads.Artifact.Path
		if libPath == "" {
			grouped := strings.Split(lib.Name, ":")
			basePath := filepath.Join(strings.Split(grouped[0], ".")...)
			name := grouped[1]
			version := grouped[2]

			libPath = filepath.Join(basePath, name, version, name+"-"+version+".jar")
		}
		existOrDownload(lib)
		cpArgs = append(cpArgs, filepath.Join(libDir, libPath))

	}
	// os.Exit(0)
	// finally append the minecraft.jar
	jarTarget := instr.Jar
	if jarTarget == "" {
		jarTarget = instr.Assets
	}
	mcJar := filepath.Join(globalDir, "versions", jarTarget, jarTarget+".jar")
	cpArgs = append(cpArgs, mcJar)

	replacer := strings.NewReplacer(
		v("auth_player_name"), profile.Name,
		v("version_name"), jarTarget,
		v("game_directory"), cwd,
		v("assets_root"), filepath.Join(m.Directory, "assets"),
		v("assets_index_name"), instr.Assets, // asset index version
		v("auth_uuid"), profile.ID, // profile id
		v("auth_access_token"), creds.AccessToken,
		v("user_type"), "mojang", // unsure about this one (legacy mc login flag?)
		v("version_type"), instr.Type, // release / snapshot … etc
	)

	args := replacer.Replace(instr.LaunchArgs())

	cmdArgs := []string{
		"-Xss1M",
		"-Djava.library.path=" + tmpDir,
		"-Dminecraft.launcher.brand=minepkg",
		// "-Dminecraft.launcher.version=" + "0.0.2", // TODO: implement!
		"-Dminecraft.client.jar=" + mcJar,
		"-cp",
		strings.Join(cpArgs, ":"),
		// "-Xmx2G", // TODO: option!
		// "-XX:+UnlockExperimentalVMOptions",
		// "-XX:+UseG1GC",
		// "-XX:G1NewSizePercent=20",
		// "-XX:G1ReservePercent=20",
		// "-XX:MaxGCPauseMillis=50",
		// "-XX:G1HeapRegionSize=32M",
		instr.MainClass,
	}
	cmdArgs = append(cmdArgs, strings.Split(args, " ")...)

	// fmt.Println("cmd: ")
	// fmt.Println(cmdArgs)
	// fmt.Println("tmpdir: + " + tmpDir)
	// os.Exit(0)

	cmd := exec.Command("java", cmdArgs...)

	// TODO: detatch from process
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()

	if err != nil {
		return err
	}
	return nil
}

func existOrDownload(lib lib) {
	home, _ := homedir.Dir()
	globalDir := filepath.Join(home, ".minepkg/libraries")
	path := filepath.Join(globalDir, lib.Filepath())
	url := lib.DownloadURL()
	if lib.Natives[runtime.GOOS] != "" {
		nativeID, _ := lib.Natives[runtime.GOOS]
		native := lib.Downloads.Classifiers[nativeID]
		url = native.URL
		path = filepath.Join(globalDir, native.Path)
	}
	if _, err := os.Stat(path); err == nil {
		return
	}

	res, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	if res.StatusCode != http.StatusOK {
		panic(url + " did not return status code 200")
	}
	fmt.Println("downloading: " + path)
	// create directory first
	os.MkdirAll(filepath.Dir(path), 0755)
	// file next
	target, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(target, res.Body)
	if err != nil {
		panic(err)
	}
}

func extractNative(jar string, target string) error {
	r, err := zip.OpenReader(jar)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		// skip META-INF dir
		if strings.HasPrefix(f.Name, "META-INF") {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		f, err := os.Create(filepath.Join(target, f.Name))
		if err != nil {
			return err
		}

		io.Copy(f, rc)
		rc.Close()
	}
	return nil
}

// v for variable
func v(s string) string {
	return "${" + s + "}"
}

func (m *McInstance) launchManifest() (*mcLaunchManifest, error) {
	manifest := m.Manifest
	// TODO: this is just for demo. make it work with anything else than fabric
	switch {
	case manifest.Requirements.Fabric != "":
		return m.resolveFabricManifest()
	case manifest.Requirements.Forge != "":
		panic("Forge is not supported")
	default:
		return m.getLaunchManifest(manifest.Requirements.Minecraft)
	}
}

func (m *McInstance) getLaunchManifest(v string) (*mcLaunchManifest, error) {
	buf, err := ioutil.ReadFile(filepath.Join(m.Directory, "versions", v, v+".json"))
	if err != nil {
		return m.fetchVanillaManifest(v)
		// return nil, err
	}
	instructions := mcLaunchManifest{}
	json.Unmarshal(buf, &instructions)
	return &instructions, nil
}

func (m *McInstance) resolveFabricManifest() (*mcLaunchManifest, error) {
	// TODO: Minecraft is a range, not a version number
	matched, err := getFabricLoaderForGameVersion(m.Manifest.Requirements.Minecraft)
	if err != nil {
		return nil, err
	}

	loader := matched.Loader.Version
	mappings := matched.Mappings.Version
	man, err := m.fetchFabricManifest(loader, mappings)
	if err != nil {
		return nil, err
	}

	return man, nil
}

func (m *McInstance) fetchFabricManifest(loader string, mappings string) (*mcLaunchManifest, error) {
	manifest := mcLaunchManifest{}
	res, err := http.Get("https://fabricmc.net/download/vanilla?format=profileJson&loader=" + url.QueryEscape(loader) + "&yarn=" + url.QueryEscape(mappings))
	if err != nil {
		return nil, err
	}

	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	version := m.Manifest.Requirements.Minecraft + "-fabric-" + loader
	dir := filepath.Join(m.Directory, "versions", m.Manifest.Requirements.Minecraft+"-fabric-"+loader)
	os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return nil, err
	}
	ioutil.WriteFile(filepath.Join(dir, version+".json"), buf, 0666)

	if err = json.Unmarshal(buf, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func (m *McInstance) fetchVanillaManifest(version string) (*mcLaunchManifest, error) {
	mcVersions, err := GetMinecraftReleases(context.TODO())
	if err != nil {
		return nil, err
	}

	manifestURL := ""
	for _, mc := range mcVersions.Versions {
		if mc.ID == version {
			manifestURL = mc.URL
		}
	}
	if manifestURL == "" {
		return nil, ErrorNoVersion
	}

	manifest := mcLaunchManifest{}
	res, err := http.Get(manifestURL)
	if err != nil {
		return nil, err
	}

	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(m.Directory, "versions", version)
	os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return nil, err
	}
	ioutil.WriteFile(filepath.Join(dir, version+".json"), buf, 0666)

	if err = json.Unmarshal(buf, &manifest); err != nil {
		return nil, err
	}

	// TODO: this is a side effect. it should not be here
	jarRes, err := http.Get(manifest.Downloads.Client.URL)
	if err != nil {
		return nil, err
	}
	jarDest, err := os.Create(filepath.Join(dir, version+".jar"))
	if err != nil {
		return nil, err
	}

	// copy the jar
	if _, err = io.Copy(jarDest, jarRes.Body); err != nil {
		return nil, err
	}

	return &manifest, nil
}

func (m *McInstance) ensureAssets(man *mcLaunchManifest) error {
	assets := mcAssetsIndex{}

	assetJSONPath := filepath.Join(m.Directory, "assets/indexes", man.Assets+".json")
	buf, err := ioutil.ReadFile(assetJSONPath)
	if err != nil {
		res, err := http.Get(man.AssetIndex.URL)
		if err != nil {
			return err
		}

		buf, err = ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}

		os.MkdirAll(filepath.Join(m.Directory, "assets/indexes"), os.ModePerm)
		err = ioutil.WriteFile(assetJSONPath, buf, 0666)
		if err != nil {
			return err
		}
	}
	json.Unmarshal(buf, &assets)

	for _, asset := range assets.Objects {
		file := filepath.Join(m.Directory, "assets/objects", asset.UnixPath())
		if _, err := os.Stat(file); os.IsNotExist(err) {
			fmt.Println("downloading: " + asset.DownloadURL())
			fileRes, err := http.Get(asset.DownloadURL())
			// TODO: check status code and all the things!
			os.MkdirAll(filepath.Join(m.Directory, "assets/objects", asset.Hash[:2]), os.ModePerm)
			dest, err := os.Create(file)
			if err != nil {
				return err
			}
			_, err = io.Copy(dest, fileRes.Body)
			if err != nil {
				return err
			}
		}
	}

	fmt.Println("ensuring assets are there …")
	fmt.Println("for: " + man.Assets)
	return nil
}

func (m *McInstance) verstionToLaunch() (string, error) {

	if m.Manifest.Requirements.Fabric != "" {
		fmt.Println("YAY FABRIC")
		return "fabric-loader-0.4.6+build.141-1.14+build.6", nil
	}

	if m.Manifest.Requirements.Forge != "" {
		fmt.Println("forge.. nice")
		return "1.12.2-forge", nil
	}

	constraint, _ := semver.NewConstraint(m.Manifest.Requirements.Minecraft)
	versions := m.AvailableVersions()

	// find newest compatible version
	for _, v := range versions {
		if constraint.Check(v) {
			return v.String(), nil
		}
	}

	return "", ErrorNoVersion
}

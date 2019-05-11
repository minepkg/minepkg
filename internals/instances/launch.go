package instances

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
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
)

// Launch starts the minecraft instance
func (m *McInstance) Launch() error {
	home, _ := homedir.Dir()
	globalDir := filepath.Join(home, ".minepkg")
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// get the latest compatible version to our minepkg requirement
	version, err := m.verstionToLaunch()
	if err != nil {
		return err
	}

	// instructions might be a bad name. this file tells us howto construct the start command
	instr, err := m.getLaunchInstructions(version)
	if err != nil {
		return err
	}

	if instr.InheritsFrom != "" {
		parent, err := m.getLaunchInstructions(instr.InheritsFrom)
		if err != nil {
			return err
		}
		instr.MergeWith(parent)
	}

	// here we snack the login info from .minecraft
	profiles, err := m.getProfiles()
	if err != nil {
		return err
	}

	profile := profiles.lastDbEntry()
	profileID := profiles.SelectedUser.Profile
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
		jarTarget = version
	}
	mcJar := filepath.Join(globalDir, "versions", jarTarget, jarTarget+".jar")
	cpArgs = append(cpArgs, mcJar)

	replacer := strings.NewReplacer(
		v("auth_player_name"), profile.Profiles[profileID].DisplayName,
		v("version_name"), version,
		v("game_directory"), cwd,
		v("assets_root"), filepath.Join(m.Directory, "assets"),
		v("assets_index_name"), instr.Assets, // asset index version
		v("auth_uuid"), profiles.SelectedUser.Profile, // profile id
		v("auth_access_token"), profile.AccessToken,
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
		"-Xmx2G", // TODO: option!
		"-XX:+UnlockExperimentalVMOptions",
		"-XX:+UseG1GC",
		"-XX:G1NewSizePercent=20",
		"-XX:G1ReservePercent=20",
		"-XX:MaxGCPauseMillis=50",
		"-XX:G1HeapRegionSize=32M",
		instr.MainClass,
	}
	cmdArgs = append(cmdArgs, strings.Split(args, " ")...)

	// fmt.Println("final cmd: ")
	// fmt.Println(cmdArgs)
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
	fmt.Println("Downloading to: " + path)
	fmt.Println("from: " + url)
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

func (m *McInstance) getProfiles() (*vanillaProfiles, error) {
	buf, err := ioutil.ReadFile(filepath.Join(m.Directory, "launcher_profiles.json"))
	if err != nil {
		return nil, err
	}
	profiles := vanillaProfiles{}
	json.Unmarshal(buf, &profiles)
	return &profiles, nil
}

func (m *McInstance) getLaunchInstructions(v string) (*launchInstructions, error) {
	buf, err := ioutil.ReadFile(filepath.Join(m.Directory, "versions", v, v+".json"))
	if err != nil {
		return nil, err
	}
	instructions := launchInstructions{}
	json.Unmarshal(buf, &instructions)
	return &instructions, nil
}

func (m *McInstance) verstionToLaunch() (string, error) {

	fmt.Println(m.Manifest)
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

type launchInstructions struct {
	// MinecraftArguments are used before 1.13 (?)
	MinecraftArguments string `json:"minecraftArguments"`
	// Arguments is the new (complicated) system
	Arguments struct {
		Game []stringArgument `json:"game"`
		JVM  []stringArgument `json:"jvm"`
	} `json:"arguments"`
	Libraries    []lib  `json:"libraries"`
	Type         string `json:"type"`
	MainClass    string `json:"mainClass"`
	Jar          string `json:"jar"`
	Assets       string `json:"assets"`
	InheritsFrom string `json:"inheritsFrom"`
}

func (l *launchInstructions) MergeWith(merge *launchInstructions) {
	l.Libraries = append(l.Libraries, merge.Libraries...)

	if l.MainClass == "" {
		l.MainClass = merge.MainClass
	}
	if l.Assets == "" {
		l.Assets = merge.Assets
	}

	if len(l.Arguments.Game) == 0 {
		l.Arguments = merge.Arguments
	}
}

func (l *launchInstructions) LaunchArgs() string {
	// easy minecraft versions before 1.13
	if l.MinecraftArguments != "" {
		return l.MinecraftArguments
	}

	// TODO: this is not a full implementation
	args := make([]string, 0)
	for _, arg := range l.Arguments.Game {
		// pretty bad, we just skip all rules here
		if len(arg.Rules) != 0 {
			continue
		}
		args = append(args, strings.Join(arg.Value, ""))
	}

	return strings.Join(args, " ")
}

type argument struct {
	// Value is the actual argument
	Value stringSlice `json:"value"`
	Rules []libRule   `json:"rules"`
}

type stringSlice []string

func (w *stringSlice) String() string {
	return strings.Join(*w, " ")
}

// UnmarshalJSON is needed because argument sometimes is a string
func (w *stringSlice) UnmarshalJSON(data []byte) (err error) {
	var arg []string

	if string(data[0]) == "[" {
		err := json.Unmarshal(data, &arg)
		if err != nil {
			return err
		}
		*w = arg
	}

	*w = []string{string(data)}
	return nil
}

type stringArgument struct{ argument }

// UnmarshalJSON is needed because argument sometimes is a string
func (w *stringArgument) UnmarshalJSON(data []byte) (err error) {
	var arg argument
	if string(data[0]) == "{" {
		err := json.Unmarshal(data, &arg)
		if err != nil {
			return err
		}
		w.argument = arg
		return nil
	}

	var str string
	err = json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	w.Value = []string{str}
	return nil
}

type lib struct {
	Name      string `json:"name"`
	Downloads struct {
		Artifact    artifact            `json:"artifact"`
		Classifiers map[string]artifact `json:"classifiers"`
	} `json:"downloads,omitempty"`
	URL     string            `json:"url"`
	Rules   []libRule         `json:"rules"`
	Natives map[string]string `json:"natives"`
}

func (l *lib) Filepath() string {
	libPath := l.Downloads.Artifact.Path
	if libPath == "" {
		grouped := strings.Split(l.Name, ":")
		basePath := filepath.Join(strings.Split(grouped[0], ".")...)
		name := grouped[1]
		version := grouped[2]

		libPath = filepath.Join(basePath, name, version, name+"-"+version+".jar")
	}
	return libPath
}

func (l *lib) DownloadURL() string {
	switch {
	case l.Downloads.Artifact.URL != "":
		return l.Downloads.Artifact.URL
	case l.URL != "":
		return l.URL + l.Filepath()
	default:
		return "https://libraries.minecraft.net/" + l.Filepath()
	}
}

type artifact struct {
	Path string      `json:"path"`
	Sha1 string      `json:"sha1"`
	Size json.Number `json:"size"`
	URL  string      `json:"url"`
}

type libRule struct {
	Action string `json:"action"`
	OS     struct {
		Name string `json:"name"`
	} `json:"os"`
}

type profile struct {
	Type     string `json:"type"`
	LastUsed string `json:"lastUsed"`
}

// lastDbEntry should be used as parameters
func (p *vanillaProfiles) lastDbEntry() dbEntry {
	last := p.SelectedUser.Account
	return p.AuthenticationDatabase[last]
}

type dbEntry struct {
	AccessToken string               `json:"accessToken"`
	Username    string               `json:"username"`
	Profiles    map[string]dbProfile `json:"profiles"`
}

type dbProfile struct {
	DisplayName string `json:"displayName"`
}

type vanillaProfiles struct {
	Profiles               map[string]profile `json:"profiles"`
	ClientToken            string             `json:"clientToken"`
	AuthenticationDatabase map[string]dbEntry `json:"authenticationDatabase"`
	SelectedUser           struct {
		Account string `json:"account"`
		Profile string `json:"profile"`
	} `json:"selectedUser"`
}

package instances

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Masterminds/semver"
)

var (
	// ErrorLaunchNotImplemented is returned if attemting to start a non vanilla instance
	ErrorLaunchNotImplemented = errors.New("Can only launch vanilla instances (for now)")
)

// Launch starts the minecraft instance
func (m *McInstance) Launch() error {
	if m.Flavour != FlavourVanilla {
		return ErrorLaunchNotImplemented
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
	libDir := filepath.Join(m.Directory, "libraries")

	// build that spooky -cp arg
	// TODO: use rules! some libs have to be excluded on osx
	var cpArgs []string
	for _, lib := range instr.Libraries {
		skip := false
		for _, rule := range lib.Rules {
			switch {
			case rule.Action == "allow" && rule.OS.Name != runtime.GOOS:
				skip = true
			case rule.Action == "disallow" && rule.OS.Name == runtime.GOOS:
				skip = true
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
			err := extractNative(filepath.Join(libDir, native.Path), tmpDir)
			if err != nil {
				return err
			}
		}
		// not skipped. append this library to our doom -cp arg
		cpArgs = append(cpArgs, filepath.Join(libDir, lib.Downloads.Artifact.Path))
	}
	// finally append the minecraft.jar
	mcJar := filepath.Join(m.Directory, "versions", version, version+".jar")
	cpArgs = append(cpArgs, mcJar)

	replacer := strings.NewReplacer(
		v("auth_player_name"), profile.Profiles[profileID].DisplayName,
		v("version_name"), version,
		v("game_directory"), m.Directory,
		v("assets_root"), filepath.Join(m.Directory, "assets"),
		v("assets_index_name"), instr.Assets, // asset index version
		v("auth_uuid"), profiles.SelectedUser.Profile, // profile id
		v("auth_access_token"), profile.AccessToken,
		v("user_type"), "mojang", // unsure about this one (legacy mc login flag?)
		v("version_type"), instr.Type, // release / snapshot … etc
	)
	args := replacer.Replace(instr.MinecraftArguments)

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
	constraint, _ := semver.NewConstraint(m.Manifest.Requirements.MinecraftVersion)
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
	MinecraftArguments string `json:"minecraftArguments"`
	Libraries          []lib  `json:"libraries"`
	Type               string `json:"type"`
	MainClass          string `json:"mainClass"`
	Assets             string `json:"assets"`
}

type lib struct {
	Name      string `json:"name"`
	Downloads struct {
		Artifact    artifact            `json:"artifact"`
		Classifiers map[string]artifact `json:"classifiers"`
	} `json:"downloads"`
	Rules   []libRule         `json:"rules"`
	Natives map[string]string `json:"natives"`
}

type artifact struct {
	Path string `json:"path"`
	Sha1 string `json:"sha1"`
	Size string `json:"size"`
	URL  string `json:"url"`
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

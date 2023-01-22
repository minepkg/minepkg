package minecraft

import (
	"encoding/json"
	"runtime"
	"strings"
)

// SupportedLauncherVersion indicates the maximum Launch Manifest version that is supported
const SupportedLauncherVersion = 21

// LaunchManifest is a version.json manifest that is used to launch minecraft instances
// Example files:
//   - Index: https://launchermeta.mojang.com/mc/game/version_manifest.json
//   - Old format: https://launchermeta.mojang.com/v1/packages/cb32af124abf1bc87c38b788926b3e592126a77c/1.9.1.json
//   - New format: https://piston-meta.mojang.com/v1/packages/6607feafdb2f96baad9314f207277730421a8e76/1.19.3.json
type LaunchManifest struct {
	// MinecraftArguments are used before 1.13 (?)
	// It is a string that only contains the arguments to be passed to Minecraft (no JVM arguments).
	// It is replaced by the Arguments field in 1.13
	MinecraftArguments string `json:"minecraftArguments"`
	// Arguments is an object that contains the arguments to be passed to Minecraft and the JVM.
	// It is used in 1.13 and newer.
	Arguments *Arguments `json:"arguments,omitempty"`
	// MainClass is the main class to launch (eg. net.minecraft.client.main.Main) – modded versions (like fabric) usually override this
	MainClass string `json:"mainClass"`

	// Downloads contains the client and server artifacts (jar files)
	// Newer versions also contain the mappings txt files (client_mappings and server_mappings)
	Downloads *struct {
		// Client is the main client jar file ("minecraft.jar")
		Client         Artifact `json:"client"`
		Server         Artifact `json:"server"`
		ClientMappings Artifact `json:"client_mappings,omitempty"`
		ServerMappings Artifact `json:"server_mappings,omitempty"`
	} `json:"downloads"`
	// Libraries is a list of libraries that are required to launch the game.
	// They need to be downloaded and added to the classpath.
	// Libraries can have rules that are used to determine if they should be applied.
	Libraries []Library `json:"libraries"`

	// JavaVersion is the version of the Java Runtime that is required to launch the game.
	JavaVersion *struct {
		Component    string `json:"component"`    // The official launcher uses these names (they are not that useful)
		MajorVersion int    `json:"majorVersion"` // Java Version number required (eg. 16, 17)
	} `json:"javaVersion"`
	// Assets is the ID of the assets index (eg. 1.16)
	Assets string `json:"assets"`
	// AssetIndex is some metadata about the assets index file
	// The assets index is a json file that contains a list of all assets (textures, sounds, etc.)
	// The "URL" can be fetched and unmarshalled into an AssetIndex struct
	AssetIndex struct {
		ID        string `json:"id"`
		Sha1      string `json:"sha1"`
		Size      int    `json:"size"`
		TotalSize int    `json:"totalSize"`
		URL       string `json:"url"`
	} `json:"assetIndex,omitempty"`

	// ID is the version number (eg. 1.16.5)
	ID string `json:"id"`
	// InheritsFrom is used by modded versions to inherit from a vanilla version
	InheritsFrom string `json:"inheritsFrom"`
	// Type is the type of the version (eg. release, snapshot, old_alpha, old_beta)
	Type string `json:"type"`
	// unsure what this is, might be a modding thing
	Jar string `json:"jar"`
	// MinimumLauncherVersion is the minimum launcher version required to launch this version.
	// This indicates the version of the launch manifest.
	MinimumLauncherVersion int `json:"minimumLauncherVersion"`
}

type Arguments struct {
	Game []Argument `json:"game"`
	JVM  []Argument `json:"jvm"`
}

// Argument is slice of command values that can be applied to the JVM or Game
// they can have rules that are used to determine if they should be applied
// Example:
//
//	{
//		"value": "-Xss1M"
//		"rules": [{
//			"action": "allow",
//			"os": { "arch": "x86" }
//		}]
//	}
type Argument struct{ ActualArgument }

type ActualArgument struct {
	// Value is the actual argument
	Value stringSlice `json:"value"`
	Rules []Rule      `json:"rules"`
}

// Applies returns true if every [Rule] in this argument applies (to this OS).
func (a *Argument) Applies() bool {
	for _, rule := range a.Rules {
		if !rule.Applies() {
			return false
		}
	}
	return true
}

// UnmarshalJSON is needed because argument sometimes is a string
func (a *Argument) UnmarshalJSON(data []byte) (err error) {
	// looks like an object
	if string(data[0]) == "{" {
		// unmarshal ignoring the UnmarshalJSON method

		err := json.Unmarshal(data, &a.ActualArgument)
		if err != nil {
			return err
		}

		return nil
	}

	// looks like a string, wrap it in an argument object
	var str string
	err = json.Unmarshal(data, &str)
	if err != nil {
		return err
	}

	// set it as the value
	a.ActualArgument.Value = []string{str}
	return nil
}

// JarName returns this manifests jar name (for example `1.12.0.jar`)
func (l *LaunchManifest) JarName() string {
	return l.MinecraftVersion() + ".jar"
}

// MinecraftVersion returns the minecraft version
func (l *LaunchManifest) MinecraftVersion() string {
	v := l.Jar
	if v == "" {
		v = l.InheritsFrom
	}
	if v == "" {
		v = l.ID
	}
	return v
}

// JVMArgs returns the jvm arguments that apply to this OS as a slice of strings
// eg ["-Xmx1G", "-Xms1G", "-Djava.library.path=${natives_directory]"]
// may contain variables that need to be replaced
// note that this can be an empty slice (eg for 1.12 or older)
func (l *LaunchManifest) JVMArgs() []string {
	args := make([]string, 0, len(l.Arguments.JVM))

	for _, arg := range l.Arguments.JVM {
		if !arg.Applies() {
			continue
		}
		args = append(args, strings.Join(arg.Value, ""))
	}

	return args
}

// GameArgs returns the game arguments that apply to this OS as a slice of strings
// eg ["--username", "${auth_player_name}"]
// may contain variables that need to be replaced
func (l *LaunchManifest) GameArgs() []string {
	// minecraft versions before 1.13 and forge (?)
	if l.MinecraftArguments != "" {
		return strings.Split(strings.TrimSpace(l.MinecraftArguments), " ")
	}

	args := make([]string, 0, len(l.Arguments.Game))

	for _, arg := range l.Arguments.Game {
		if !arg.Applies() {
			continue
		}
		args = append(args, strings.Join(arg.Value, ""))
	}

	return args
}

// FullArgs returns the launch arguments defined in the manifest as a string slice.
// This is a concatenation of [LaunchManifest.JVMArgs] the MainClass and [LaunchManifest.GameArgs].
// Uses [FallbackJVMArgs] if no JVM Arguments are defined in this manifest (in "LaunchManifest.Arguments.JVM").
//
// These arguments should be able to launch the game after replacing variables when launched with a java executable.
func (l *LaunchManifest) FullArgs() []string {
	jvmArgs := l.JVMArgs()
	if len(jvmArgs) == 0 {
		jvmArgs = FallbackJVMArgs(runtime.GOOS)
	}
	gameArgs := l.GameArgs()
	args := make([]string, 0, len(jvmArgs)+len(gameArgs)+1)
	args = append(args, jvmArgs...)
	args = append(args, l.MainClass)
	args = append(args, gameArgs...)

	return args
}

// MergeManifests merges important properties from the given manifests
// by modifying the source manifest.
// It merges libraries, game args and jvm args by appending them.
// This is a simple implementation. it does not merge everything and
// does not care for duplicates.
func MergeManifests(source *LaunchManifest, manifests ...*LaunchManifest) {
	for _, new := range manifests {
		if new == nil {
			panic("can't merge with nil manifest")
		}
		source.Libraries = append(source.Libraries, new.Libraries...)
		if new.MainClass != "" {
			source.MainClass = new.MainClass
		}
		if new.Assets != "" {
			source.Assets = new.Assets
		}
		if new.AssetIndex.ID != "" {
			source.AssetIndex = new.AssetIndex
		}

		if new.Type != "" {
			source.Type = new.Type
		}

		if new.Jar != "" {
			source.Jar = new.Jar
		}

		if new.JavaVersion != nil {
			source.JavaVersion = new.JavaVersion
		}

		if new.ID != "" {
			source.ID = new.ID
		}

		if new.JavaVersion != nil {
			source.JavaVersion = new.JavaVersion
		}

		// hack
		if new.Downloads != nil {
			source.Downloads = new.Downloads
		}

		// Merge launchArgs
		if new.MinecraftArguments != "" {
			source.MinecraftArguments = new.MinecraftArguments
		}

		if new.Arguments == nil {
			return
		}
		// Merge game args
		if new.Arguments.Game != nil {
			source.Arguments.Game = append(source.Arguments.Game, new.Arguments.Game...)
		}
		// Merge jvm args
		if new.Arguments.JVM != nil {
			source.Arguments.JVM = append(source.Arguments.JVM, new.Arguments.JVM...)
		}
	}
}

// FallbackJVMArgs returns some default jvm arguments for the given OS.
// This can be used if no jvm args are defined in the manifest.
// Old versions of minecraft do not define any jvm args.
//
// Note that this contains the following variables that need to be replaced:
//   - ${natives_directory}
//   - ${classpath}
//   - ${launcher_name}
//   - ${launcher_version}
func FallbackJVMArgs(os string) []string {
	args := []string{
		"-Xms512m",
		"-Djava.library.path=${natives_directory}",
		"-Dminecraft.launcher.brand=${launcher_name}",
		"-Dminecraft.launcher.version=${launcher_version}",
	}

	if os == "windows" {
		args = append(args, "-XX:HeapDumpPath=MojangTricksIntelDriversForPerformance_javaw.exe_minecraft.exe.heapdump")
	}

	if os == "darwin" {
		args = append(args, "-XstartOnFirstThread")
	}

	args = append(args, "-cp", "${classpath}")

	return args
}

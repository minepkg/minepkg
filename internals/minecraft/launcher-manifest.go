package minecraft

import (
	"runtime"
	"strings"
)

// SupportedLauncherVersion indicates the maximum Launch Manifest version that is supported
const SupportedLauncherVersion = 21

// LaunchManifest is a version.json manifest that is used to launch minecraft instances
type LaunchManifest struct {
	// MinecraftArguments are used before 1.13 (?)
	MinecraftArguments string `json:"minecraftArguments"`
	// Arguments is the new (complicated) system
	Arguments *Arguments `json:"arguments,omitempty"`
	// MainClass is the main class to launch (eg. net.minecraft.client.main.Main) – usually overwritten by a loader like fabric
	MainClass string `json:"mainClass"`
	Downloads *struct {
		Client Artifact `json:"client"`
		Server Artifact `json:"server"`
	} `json:"downloads"`
	Libraries   []Library `json:"libraries"`
	JavaVersion *struct {
		Component    string `json:"component"`    // "java-runtime-beta" currently not used
		MajorVersion int    `json:"majorVersion"` // number like 16 or 17
	} `json:"javaVersion"`
	Type       string `json:"type"`
	Jar        string `json:"jar"`
	Assets     string `json:"assets"`
	AssetIndex struct {
		ID        string `json:"id"`
		Sha1      string `json:"sha1"`
		Size      int    `json:"size"`
		TotalSize int    `json:"totalSize"`
		URL       string `json:"url"`
	} `json:"assetIndex,omitempty"`
	InheritsFrom           string `json:"inheritsFrom"`
	ID                     string `json:"id"`
	MinimumLauncherVersion int    `json:"minimumLauncherVersion"`
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
type Argument struct {
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

// FullArgs returns the launch arguments defined in the manifest as a string slice
// this is a concatenation of JVMArgs, MainClass and GameArgs
// uses DefaultJVMArgs() if no JVMArgs are defined.
// These args should be able to launch the game after replacing variables.
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

// RequiredLibraries returns a slice of libraries that are required to launch the game (depending on the OS)
func (l *LaunchManifest) RequiredLibraries() []Library {
	libs := make([]Library, 0, len(l.Libraries))
	for _, lib := range l.Libraries {
		if lib.Applies() {
			libs = append(libs, lib)
		}
	}
	return libs
}

// MergeManifests merges important properties from the given manifests
// by modifying the source manifest.
// It merges libraries, game args and jvm args by appending them.
// This is a simple implementation. it does not merge everything and
// does not care for duplicates
func MergeManifests(source *LaunchManifest, manifests ...*LaunchManifest) {
	for _, new := range manifests {
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

		if new.Arguments != nil && len(new.Arguments.Game) != 0 {
			source.Arguments = new.Arguments
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

		if source.Arguments == nil {
			return
		}
		// Merge game args
		source.Arguments.Game = append(source.Arguments.Game, new.Arguments.Game...)
		// Merge jvm args
		source.Arguments.JVM = append(source.Arguments.JVM, new.Arguments.JVM...)
	}
}

// FallbackJVMArgs returns some default jvm arguments for the given OS
// this can be used if no jvm args are defined in the manifest
// old versions of minecraft do not define any jvm args.
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
		"-cp",
		"${classpath}",
	}

	if os == "windows" {
		args = append(args, "-XX:HeapDumpPath=MojangTricksIntelDriversForPerformance_javaw.exe_minecraft.exe.heapdump")
	}

	if os == "darwin" {
		args = append(args, "-XstartOnFirstThread")
	}

	return args
}

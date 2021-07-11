package launcher

import (
	"os/exec"

	"github.com/minepkg/minepkg/internals/instances"
	"github.com/minepkg/minepkg/internals/java"
	"github.com/minepkg/minepkg/internals/minecraft"
)

// Launcher can launch minepkg instances with CLI output
type Launcher struct {
	// Instance is the minepkg instance to be launched
	Instance *instances.Instance

	// MinepkgVersion is the version number of minepkg
	MinepkgVersion string

	Cmd *exec.Cmd
	// ServerMode indicated if this instance should be started as a server
	ServerMode bool
	// OfflineMode indicates if this server should be started in offline mode
	OfflineMode bool

	// ForceUpdate will force a full dependency resolve if set to true
	ForceUpdate bool

	// LaunchManifest is a minecraft launcher manifest. it should be set after
	// calling `Prepare`
	LaunchManifest *minecraft.LaunchManifest

	// NonInteractive determines if fancy spinners or prompts should be displayed
	NonInteractive bool

	// UseSystemJava sets the instance to use the system java
	// instead of the internal installation. This skips downloading java
	UseSystemJava bool

	// JavaVersion is the version to use
	JavaVersion string

	javaFactoryInstance *java.Factory
	java                *java.Java
	introPrinted        bool
	originalServerProps []byte
}

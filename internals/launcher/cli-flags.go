package launcher

import (
	"fmt"

	"github.com/spf13/cobra"
)

// OverwriteFlags are cli flags used to overwrite launch behavior
type OverwriteFlags struct {
	McVersion        string
	FabricVersion    string
	ForgeVersion     string
	MinepkgCompanion string
	Java             string
	Ram              int
}

func CmdOverwriteFlags(cmd *cobra.Command) *OverwriteFlags {
	flags := OverwriteFlags{}
	cmd.Flags().StringVarP(&flags.McVersion, "minecraft", "m", "", "Overwrite the required Minecraft version")
	cmd.Flags().StringVar(&flags.FabricVersion, "fabricLoader", "", "Overwrite the required fabricLoader version")
	cmd.Flags().StringVar(&flags.MinepkgCompanion, "minepkgCompanion", "", "Overwrite the required minepkg companion version (can also be \"none\")")
	cmd.Flags().IntVar(&flags.Ram, "ram", 0, "Overwrite the amount of RAM in MiB to use")
	cmd.Flags().StringVar(&flags.Java, "java", "", "Overwrite the Java runtime. Examples: 16-jre, 8-jre-openj9, system")

	return &flags
}

func (l *Launcher) ApplyOverWrites(o *OverwriteFlags) {
	instance := l.Instance
	if o.FabricVersion != "" {
		instance.Manifest.Requirements.FabricLoader = o.FabricVersion
	}
	if o.McVersion != "" {
		fmt.Println("Minecraft version overwritten to version: " + o.McVersion)
		instance.Manifest.Requirements.Minecraft = o.McVersion
	}
	if o.MinepkgCompanion != "" {
		fmt.Println("Companion overwritten!")
		instance.Manifest.Requirements.MinepkgCompanion = o.MinepkgCompanion
	}

	if o.Java == "system" {
		l.UseSystemJava = true
	} else {
		l.JavaVersion = o.Java
	}
}

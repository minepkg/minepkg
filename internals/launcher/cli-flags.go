package launcher

import (
	"fmt"

	"github.com/minepkg/minepkg/internals/instances"
	"github.com/spf13/cobra"
)

// OverwriteFlags are cli flags used to overwrite launch behavior
type OverwriteFlags struct {
	McVersion        string
	FabricVersion    string
	ForgeVersion     string
	MinepkgCompanion string
	Ram              int
}

func CmdOverwriteFlags(cmd *cobra.Command) *OverwriteFlags {
	flags := OverwriteFlags{}
	cmd.Flags().StringVarP(&flags.McVersion, "minecraft", "m", "", "Overwrite the required Minecraft version")
	cmd.Flags().StringVar(&flags.FabricVersion, "fabricLoader", "", "Overwrite the required fabricLoader version")
	cmd.Flags().StringVar(&flags.MinepkgCompanion, "minepkgCompanion", "", "Overwrite the required minepkg companion version (can also be \"none\")")
	cmd.Flags().IntVar(&flags.Ram, "ram", 0, "Overwrite the amount of RAM in MiB to use")

	return &flags
}

func ApplyInstanceOverwrites(instance *instances.Instance, o *OverwriteFlags) {
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
}

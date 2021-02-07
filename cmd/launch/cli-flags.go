package launch

import (
	"fmt"

	"github.com/fiws/minepkg/internals/instances"
	"github.com/spf13/cobra"
)

// OverwriteFlags are cli flags used to overwrite launch behaviour
type OverwriteFlags struct {
	McVersion        string
	FabricVersion    string
	ForgeVersion     string
	MinepkgCompanion string
}

func CmdOverwriteFlags(cmd *cobra.Command) *OverwriteFlags {
	flags := OverwriteFlags{}
	cmd.Flags().StringVarP(&flags.McVersion, "minecraft", "m", "", "Overwrite the required minepkg companion version (can also be \"none\")")
	cmd.Flags().StringVarP(&flags.FabricVersion, "fabric", "", "", "Overwrite the required fabric version")
	cmd.Flags().StringVarP(&flags.MinepkgCompanion, "minepkgCompanion", "", "", "Overwrite the required minepkg companion version")

	return &flags
}

func ApplyInstanceOverwrites(instance *instances.Instance, o *OverwriteFlags) {
	if o.FabricVersion != "" {
		instance.Manifest.Requirements.Fabric = o.FabricVersion
	}
	if o.McVersion != "" {
		fmt.Println("mc version overwritten!")
		instance.Manifest.Requirements.Minecraft = o.McVersion
	}
	if o.MinepkgCompanion != "" {
		fmt.Println("companion overwritten!")
		instance.Manifest.Requirements.MinepkgCompanion = o.MinepkgCompanion
	}
}
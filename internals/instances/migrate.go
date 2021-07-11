package instances

import (
	"fmt"
	"os"
)

func (i *Instance) migrate() error {
	if err := i.migrateManifest(); err != nil {
		return err
	}

	if err := i.migrateLockfile(); err != nil {
		return err
	}

	return nil
}

func (i *Instance) migrateManifest() error {
	m := i.Manifest
	migrated := false
	if m.Requirements.Fabric != "" {
		m.Requirements.FabricLoader = m.Requirements.Fabric
		m.Requirements.Fabric = ""
		migrated = true
	}
	if m.Requirements.Forge != "" {
		m.Requirements.FabricLoader = m.Requirements.Forge
		m.Requirements.Forge = ""
		migrated = true
	}
	if migrated {
		if err := i.SaveManifest(); err != nil {
			return fmt.Errorf("could not save migrated manifest: %w", err)
		}
	}
	return nil
}

func (i *Instance) migrateLockfile() error {
	if i.lockfileNeedsRenameMigration {
		fmt.Println("migrating lockfile")
		return os.Rename(i.legacyLockfilePath(), i.LockfilePath())
	}

	fmt.Println("ignored")

	return nil
}

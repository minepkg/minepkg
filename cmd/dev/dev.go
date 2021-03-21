package dev

import (
	"github.com/spf13/cobra"
)

var SubCmd = &cobra.Command{
	Use:   "dev",
	Short: "Advanced package dev related tasks (eg. build)",
}

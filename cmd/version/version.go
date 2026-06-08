package version

import (
	"fmt"
	"github.com/spf13/cobra"
)

var Version = "dev"
var Commit = "none"

var Cmd = &cobra.Command{
	Use:   "version",
	Short: "Print RawVelo version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("RawVelo %s (commit: %s)\n", Version, Commit)
	},
}

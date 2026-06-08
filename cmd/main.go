package main

import (
	"os"
	"rawvelo/cmd/dump"
	"rawvelo/cmd/iface"
	"rawvelo/cmd/ping"
	"rawvelo/cmd/run"
	"rawvelo/cmd/secret"
	"rawvelo/cmd/version"
	"rawvelo/internal/flog"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "rawvelo",
	Short: "RawVelo — Raw-socket KCP tunnel.",
	Long:  `RawVelo is a bidirectional packet-level proxy using KCP and raw socket transport with encryption.`,
}

func main() {
	rootCmd.AddCommand(run.Cmd)
	rootCmd.AddCommand(dump.Cmd)
	rootCmd.AddCommand(ping.Cmd)
	rootCmd.AddCommand(secret.Cmd)
	rootCmd.AddCommand(iface.Cmd)
	rootCmd.AddCommand(version.Cmd)

	if err := rootCmd.Execute(); err != nil {
		flog.Errorf("%v", err)
		os.Exit(1)
	}
}

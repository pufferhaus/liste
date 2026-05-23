package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	buildVersion = "dev"
	buildCommit  = "none"
	buildDate    = "unknown"
)

// SetVersionInfo is called from main to inject build-time values.
func SetVersionInfo(version, commit, date string) {
	buildVersion = version
	buildCommit = commit
	buildDate = date
	rootCmd.Version = version
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Args:  cobra.NoArgs,
	RunE:  runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersion(cmd *cobra.Command, args []string) error {
	if !flagJSON && !flagQuiet {
		printBanner(buildVersion)
	}
	if flagJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]string{
			"version": buildVersion,
			"commit":  buildCommit,
			"date":    buildDate,
		})
		return nil
	}

	fmt.Fprintf(os.Stdout, "liste %s (commit: %s, built: %s)\n", buildVersion, buildCommit, buildDate)
	return nil
}

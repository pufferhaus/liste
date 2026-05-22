package cmd

import (
	"fmt"
	"os"

	"github.com/pufferhaus/liste/internal/discovery"
	"github.com/pufferhaus/liste/internal/output"
	"github.com/pufferhaus/liste/internal/store"
	"github.com/pufferhaus/liste/internal/tui"
	"github.com/spf13/cobra"
)

var (
	flagJSON        bool
	flagQuiet       bool
	flagProject     string
	flagInteractive bool
)

var rootCmd = &cobra.Command{
	Use:   "liste",
	Short: "Portable roadmap and project tracker",
	Long:  "A CLI tool for managing project roadmaps as markdown files. Designed for both humans and AI agents.",
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVar(&flagQuiet, "quiet", false, "Minimal output (IDs only)")
	rootCmd.PersistentFlags().StringVarP(&flagProject, "project", "p", "", "Target a specific sub-project")
	rootCmd.PersistentFlags().BoolVarP(&flagInteractive, "interactive", "i", false, "Launch interactive TUI")

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if !flagInteractive {
			return nil
		}
		if flagJSON || flagQuiet {
			return fmt.Errorf("--interactive cannot be used with --json or --quiet")
		}
		result, err := getDiscovery()
		if err != nil {
			return err
		}
		rootStore := store.New(result.Root)
		cfg, err := rootStore.ReadConfig()
		if err != nil {
			return err
		}
		if err := tui.Run(result, cfg); err != nil {
			return err
		}
		os.Exit(0)
		return nil
	}
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// getFormatter returns a formatter based on current flags.
func getFormatter() *output.Formatter {
	format := output.FormatTable
	if flagJSON {
		format = output.FormatJSON
	} else if flagQuiet {
		format = output.FormatQuiet
	}
	return output.New(os.Stdout, format)
}

// getStore resolves the store for the current context (CWD + project flag).
func getStore() (*store.Store, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}

	result, err := discovery.Discover(cwd)
	if err != nil {
		return nil, fmt.Errorf("discovering projects: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("no .liste/ found (run 'liste init' to create one)")
	}

	if flagProject != "" {
		s := discovery.StoreForProject(result, flagProject)
		if s == nil {
			return nil, fmt.Errorf("project %q not found", flagProject)
		}
		return s, nil
	}

	return store.New(result.Root), nil
}

// getDiscovery returns the full discovery result for multi-project commands.
func getDiscovery() (*discovery.Result, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}

	result, err := discovery.Discover(cwd)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("no .liste/ found (run 'liste init' to create one)")
	}

	return result, nil
}

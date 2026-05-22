package cmd

import (
	"github.com/pufferhaus/liste/internal/output"
	"github.com/pufferhaus/liste/internal/store"
	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List all discovered projects",
	Long:  "Show the root project and all discovered sub-projects.",
	Args:  cobra.NoArgs,
	RunE:  runProjects,
}

func init() {
	rootCmd.AddCommand(projectsCmd)
}

func runProjects(cmd *cobra.Command, args []string) error {
	result, err := getDiscovery()
	if err != nil {
		return err
	}

	// Get item counts for each sub-project
	var subs []output.ProjectDisplay
	for _, sub := range result.SubProjects {
		s := store.New(sub.Path)
		items, _ := s.ListItems()
		subs = append(subs, output.ProjectDisplay{
			Name:      sub.Name,
			Path:      sub.Path,
			ItemCount: len(items),
		})
	}

	// Get root info
	rootStore := store.New(result.Root)
	cfg, _ := rootStore.ReadConfig()
	rootName := result.Root
	if cfg != nil {
		rootName = cfg.Project
	}

	f := getFormatter()
	f.ProjectList(rootName, subs)
	return nil
}

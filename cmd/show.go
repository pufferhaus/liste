package cmd

import (
	"strings"

	"github.com/pufferhaus/liste/internal/output"
	"github.com/pufferhaus/liste/internal/resolver"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show full item detail",
	Long:  "Display all metadata, links, and body content for an item.",
	Args:  cobra.ExactArgs(1),
	RunE:  runShow,
}

func init() {
	rootCmd.AddCommand(showCmd)
}

func runShow(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	id := strings.ToUpper(args[0])
	item, err := s.ReadItem(id)
	if err != nil {
		return err
	}

	// Resolve inverse links
	inverse, _ := resolver.ResolveInverse(s, id)
	var inverseDisplay []output.InverseLinkDisplay
	for _, inv := range inverse {
		inverseDisplay = append(inverseDisplay, output.InverseLinkDisplay{
			Type:     string(inv.Type),
			SourceID: inv.SourceID,
		})
	}

	f := getFormatter()
	f.ItemDetail(item, inverseDisplay)
	return nil
}

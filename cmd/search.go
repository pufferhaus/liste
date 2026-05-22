package cmd

import (
	"strings"

	"github.com/pufferhaus/liste/internal/model"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search items by title and body content",
	Long:  "Full-text search across all items in the current project.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	query := strings.ToLower(strings.Join(args, " "))

	items, err := s.ListItems()
	if err != nil {
		return err
	}

	var matches []*model.Item
	for _, item := range items {
		if matchesQuery(item, query) {
			matches = append(matches, item)
		}
	}

	f := getFormatter()
	f.ItemList(matches)
	return nil
}

func matchesQuery(item *model.Item, query string) bool {
	// Search in title
	if strings.Contains(strings.ToLower(item.Title), query) {
		return true
	}
	// Search in body
	if strings.Contains(strings.ToLower(item.Body), query) {
		return true
	}
	// Search in ID
	if strings.Contains(strings.ToLower(item.ID), query) {
		return true
	}
	// Search in tags
	for _, tag := range item.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	return false
}

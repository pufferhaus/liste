package cmd

import (
	"github.com/pufferhaus/liste/internal/model"
	"github.com/spf13/cobra"
)

var (
	listStatus   string
	listType     string
	listPriority string
	listTag      string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List items",
	Long:  "List items with optional filters for status, type, priority, and tag.",
	Args:  cobra.NoArgs,
	RunE:  runList,
}

func init() {
	listCmd.Flags().StringVar(&listStatus, "status", "", "Filter by status")
	listCmd.Flags().StringVar(&listType, "type", "", "Filter by type")
	listCmd.Flags().StringVar(&listPriority, "priority", "", "Filter by priority")
	listCmd.Flags().StringVar(&listTag, "tag", "", "Filter by tag")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	items, err := s.ListItems()
	if err != nil {
		return err
	}

	// Apply filters
	filtered := make([]*model.Item, 0, len(items))
	for _, item := range items {
		if listStatus != "" && item.Status != listStatus {
			continue
		}
		if listType != "" && string(item.Type) != listType {
			continue
		}
		if listPriority != "" && item.Priority != listPriority {
			continue
		}
		if listTag != "" && !hasTag(item.Tags, listTag) {
			continue
		}
		filtered = append(filtered, item)
	}

	f := getFormatter()
	f.ItemList(filtered)
	return nil
}

func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

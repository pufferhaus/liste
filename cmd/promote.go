package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/pufferhaus/liste/internal/model"
	"github.com/spf13/cobra"
)

var promoteCmd = &cobra.Command{
	Use:   "promote <id> [new-type]",
	Short: "Promote an item to a different type",
	Long:  "Change the type of an item (e.g., idea -> feature). Assigns a new ID with the target type prefix.",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPromote,
}

func init() {
	rootCmd.AddCommand(promoteCmd)
}

func runPromote(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	id := strings.ToUpper(args[0])

	// Default promote target is "feature"
	newTypeStr := "feature"
	if len(args) > 1 {
		newTypeStr = args[1]
	}

	newType, ok := model.ParseItemType(newTypeStr)
	if !ok {
		return fmt.Errorf("invalid type %q (valid: feature, bug, idea, task, epic)", newTypeStr)
	}

	item, err := s.ReadItem(id)
	if err != nil {
		return err
	}

	if item.Type == newType {
		return fmt.Errorf("%s is already of type %s", id, newType)
	}

	// Allocate new ID
	newID, err := s.NextID(newType)
	if err != nil {
		return err
	}

	// Delete old file
	if err := s.DeleteItem(id); err != nil {
		return err
	}

	// Update item
	oldID := item.ID
	item.ID = newID
	item.Type = newType
	item.Updated = time.Now()

	// If it was an idea, move to planned
	if item.Status == "idea" {
		item.Status = "planned"
	}

	if err := s.WriteItem(item); err != nil {
		return err
	}

	f := getFormatter()
	f.Message(fmt.Sprintf("Promoted %s -> %s (%s): %s", oldID, newID, newType, item.Title))
	return nil
}

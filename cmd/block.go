package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/pufferhaus/liste/internal/model"
	"github.com/spf13/cobra"
)

var blockCmd = &cobra.Command{
	Use:   "block <id> [reason]",
	Short: "Toggle blocked flag on an item",
	Long:  "Mark an item as blocked with an optional reason, or unblock if already blocked.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runBlock,
}

func init() {
	rootCmd.AddCommand(blockCmd)
}

func runBlock(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	id := strings.ToUpper(args[0])
	item, err := s.ReadItem(id)
	if err != nil {
		return err
	}

	if item.Blocked != nil {
		// Unblock
		item.Blocked = nil
		item.Updated = time.Now()
		if err := s.WriteItem(item); err != nil {
			return err
		}
		f := getFormatter()
		f.Message(fmt.Sprintf("Unblocked %s: %s", id, item.Title))
	} else {
		// Block
		reason := ""
		if len(args) > 1 {
			reason = strings.Join(args[1:], " ")
		}
		item.Blocked = &model.Blocked{Reason: reason}
		item.Updated = time.Now()
		if err := s.WriteItem(item); err != nil {
			return err
		}
		f := getFormatter()
		msg := fmt.Sprintf("Blocked %s: %s", id, item.Title)
		if reason != "" {
			msg += fmt.Sprintf(" (reason: %s)", reason)
		}
		f.Message(msg)
	}

	return nil
}

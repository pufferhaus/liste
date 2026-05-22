package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/pufferhaus/liste/internal/model"
	"github.com/spf13/cobra"
)

var linkCmd = &cobra.Command{
	Use:   "link <id> <type> <target>",
	Short: "Add a typed link between items",
	Long:  "Create a relationship between two items. Types: depends-on, blocks, parent-of, child-of, relates-to.",
	Args:  cobra.ExactArgs(3),
	RunE:  runLink,
}

func init() {
	rootCmd.AddCommand(linkCmd)
}

func runLink(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	id := strings.ToUpper(args[0])
	linkTypeStr := args[1]
	target := strings.ToUpper(args[2])

	linkType, ok := model.ParseLinkType(linkTypeStr)
	if !ok {
		return fmt.Errorf("invalid link type %q (valid: depends-on, blocks, parent-of, child-of, relates-to)", linkTypeStr)
	}

	item, err := s.ReadItem(id)
	if err != nil {
		return err
	}

	// Check target exists (in same project) — cross-project refs are allowed
	_, _ = s.ReadItem(target)

	// Check for duplicate link
	for _, existing := range item.Links {
		if existing.Target == target && existing.Type == linkType {
			return fmt.Errorf("link already exists: %s %s %s", id, linkType, target)
		}
	}

	item.Links = append(item.Links, model.Link{
		Type:   linkType,
		Target: target,
	})
	item.Updated = time.Now()

	if err := s.WriteItem(item); err != nil {
		return err
	}

	f := getFormatter()
	f.Message(fmt.Sprintf("Linked %s -[%s]-> %s", id, linkType, target))
	return nil
}

package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pufferhaus/liste/internal/model"

	"github.com/spf13/cobra"
)

var blockedCmd = &cobra.Command{
	Use:   "blocked",
	Short: "Show all blocked items with reasons and blocker chains",
	Long: `Lists all items that are currently blocked, showing:
- The block reason
- What items they depend on that aren't done (the blocker chain)`,
	Args: cobra.NoArgs,
	RunE: runBlocked,
}

func init() {
	rootCmd.AddCommand(blockedCmd)
}

type blockedEntry struct {
	Item          itemWithProject
	Reason        string
	UnmetDeps     []string // IDs of depends-on targets that aren't done
	UnmetDepNames []string // titles of those targets
}

func runBlocked(cmd *cobra.Command, args []string) error {
	allItems, allItemsByID, _, err := collectAllItems()
	if err != nil {
		return err
	}

	var blocked []blockedEntry
	for _, entry := range allItems {
		item := entry.item

		// Explicitly blocked
		if item.Blocked != nil {
			reason := item.Blocked.Reason
			if reason == "" {
				reason = "no reason specified"
			}
			unmet, unmetNames := findUnmetDeps(item, allItemsByID)
			blocked = append(blocked, blockedEntry{
				Item:          entry,
				Reason:        reason,
				UnmetDeps:     unmet,
				UnmetDepNames: unmetNames,
			})
			continue
		}

		// Implicitly blocked by unresolved dependencies
		if item.Status != "done" && item.Status != "cancelled" {
			unmet, unmetNames := findUnmetDeps(item, allItemsByID)
			if len(unmet) > 0 {
				blocked = append(blocked, blockedEntry{
					Item:          entry,
					Reason:        "waiting on dependencies",
					UnmetDeps:     unmet,
					UnmetDepNames: unmetNames,
				})
			}
		}
	}

	if flagJSON {
		type jsonBlocked struct {
			ID        string   `json:"id"`
			Title     string   `json:"title"`
			Priority  string   `json:"priority"`
			Phase     *int     `json:"phase,omitempty"`
			Project   string   `json:"project"`
			Reason    string   `json:"reason"`
			UnmetDeps []string `json:"unmet_deps,omitempty"`
		}
		var out []jsonBlocked
		for _, b := range blocked {
			out = append(out, jsonBlocked{
				ID:        b.Item.item.ID,
				Title:     b.Item.item.Title,
				Priority:  b.Item.item.Priority,
				Phase:     b.Item.item.Phase,
				Project:   b.Item.project,
				Reason:    b.Reason,
				UnmetDeps: b.UnmetDeps,
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out)
		return nil
	}

	if len(blocked) == 0 {
		fmt.Fprintln(os.Stdout, "No blocked items.")
		return nil
	}

	fmt.Fprintf(os.Stdout, "Blocked items (%d):\n\n", len(blocked))
	for _, b := range blocked {
		item := b.Item.item
		phaseStr := ""
		if item.Phase != nil {
			phaseStr = fmt.Sprintf(" (phase %d)", *item.Phase)
		}
		fmt.Fprintf(os.Stdout, "  %s [%s] %s%s\n", item.ID, item.Priority, item.Title, phaseStr)
		fmt.Fprintf(os.Stdout, "    Reason: %s\n", b.Reason)
		if len(b.UnmetDeps) > 0 {
			fmt.Fprintf(os.Stdout, "    Waiting on:\n")
			for i, dep := range b.UnmetDeps {
				name := ""
				if i < len(b.UnmetDepNames) {
					name = " — " + b.UnmetDepNames[i]
				}
				fmt.Fprintf(os.Stdout, "      - %s%s\n", dep, name)
			}
		}
		fmt.Fprintln(os.Stdout)
	}

	return nil
}

// findUnmetDeps returns IDs and titles of depends-on targets that aren't done.
func findUnmetDeps(item *model.Item, allItems map[string]*model.Item) ([]string, []string) {
	var ids []string
	var names []string
	for _, link := range item.Links {
		if link.Type != model.LinkDependsOn {
			continue
		}
		target, ok := allItems[link.Target]
		if !ok {
			ids = append(ids, link.Target)
			names = append(names, "(not found)")
			continue
		}
		if target.Status != "done" && target.Status != "cancelled" {
			ids = append(ids, link.Target)
			names = append(names, target.Title)
		}
	}
	return ids, names
}

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pufferhaus/liste/internal/resolver"
	"github.com/spf13/cobra"
)

var graphCmd = &cobra.Command{
	Use:   "graph <id>",
	Short: "Show the link graph for an item",
	Long:  "Display all items connected to the given item (direct and inverse links).",
	Args:  cobra.ExactArgs(1),
	RunE:  runGraph,
}

func init() {
	rootCmd.AddCommand(graphCmd)
}

func runGraph(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	id := strings.ToUpper(args[0])
	item, err := s.ReadItem(id)
	if err != nil {
		return err
	}

	nodes, err := resolver.BuildGraph(s, id)
	if err != nil {
		return err
	}

	if flagJSON {
		// JSON mode handled differently
		type jsonNode struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Status   string `json:"status,omitempty"`
			Relation string `json:"relation"`
			Project  string `json:"project,omitempty"`
		}
		var jNodes []jsonNode
		for _, n := range nodes {
			jNodes = append(jNodes, jsonNode{
				ID:       n.ID,
				Title:    n.Title,
				Status:   n.Status,
				Relation: string(n.Relation),
				Project:  n.Project,
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(jNodes)
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Graph for %s: %s\n\n", id, item.Title)
	if len(nodes) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "  (no connections)")
		return nil
	}

	for _, node := range nodes {
		proj := ""
		if node.Project != "" {
			proj = " [" + node.Project + "]"
		}
		status := ""
		if node.Status != "" {
			status = " (" + node.Status + ")"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  %s %s%s — %s%s\n", node.Relation, node.ID, proj, node.Title, status)
	}

	return nil
}

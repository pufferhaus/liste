package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pufferhaus/liste/internal/model"
	"github.com/pufferhaus/liste/internal/store"

	"github.com/spf13/cobra"
)

var (
	treeDepth int
)

var treeCmd = &cobra.Command{
	Use:   "tree <id>",
	Short: "Show the hierarchy tree for an item",
	Long: `Display the full parent/child hierarchy for an item.
Shows the item and all its children (via parent-of links) recursively,
forming a tree view useful for seeing epic/feature breakdowns.`,
	Args: cobra.ExactArgs(1),
	RunE: runTree,
}

func init() {
	treeCmd.Flags().IntVar(&treeDepth, "depth", 10, "Maximum tree depth")
	rootCmd.AddCommand(treeCmd)
}

type treeNode struct {
	Item     *model.Item
	Project  string
	Children []*treeNode
	Depth    int
}

type itemInfo struct {
	item    *model.Item
	project string
}

func runTree(cmd *cobra.Command, args []string) error {
	id := strings.ToUpper(args[0])

	result, err := getDiscovery()
	if err != nil {
		return err
	}

	// Build a map of all items and their parent-of relationships
	allItems := make(map[string]itemInfo)
	childrenOf := make(map[string][]string) // parent ID -> child IDs

	rootStore := store.New(result.Root)
	rootCfg, _ := rootStore.ReadConfig()
	rootItems, err := rootStore.ListItems()
	if err != nil {
		return err
	}
	projectName := "root"
	if rootCfg != nil {
		projectName = rootCfg.Project
	}
	for _, item := range rootItems {
		allItems[item.ID] = itemInfo{item: item, project: projectName}
		for _, link := range item.Links {
			if link.Type == model.LinkParentOf {
				childrenOf[item.ID] = append(childrenOf[item.ID], link.Target)
			}
			if link.Type == model.LinkChildOf {
				childrenOf[link.Target] = append(childrenOf[link.Target], item.ID)
			}
		}
	}

	for _, sub := range result.SubProjects {
		s := store.New(sub.Path)
		items, err := s.ListItems()
		if err != nil {
			continue
		}
		for _, item := range items {
			allItems[item.ID] = itemInfo{item: item, project: sub.Name}
			for _, link := range item.Links {
				if link.Type == model.LinkParentOf {
					childrenOf[item.ID] = append(childrenOf[item.ID], link.Target)
				}
				if link.Type == model.LinkChildOf {
					childrenOf[link.Target] = append(childrenOf[link.Target], item.ID)
				}
			}
		}
	}

	// Find the root item
	if _, ok := allItems[id]; !ok {
		return fmt.Errorf("item %s not found", id)
	}

	// Build tree recursively from requested item down
	root := buildTreeNode(id, allItems, childrenOf, 0, treeDepth)

	if flagJSON {
		renderTreeJSON(root)
		return nil
	}

	fmt.Fprintf(os.Stdout, "Tree for %s:\n\n", id)
	renderTreeText(root, "", true)
	return nil
}

func buildTreeNode(id string, allItems map[string]itemInfo, childrenOf map[string][]string, depth, maxDepth int) *treeNode {
	info, ok := allItems[id]
	if !ok {
		return &treeNode{
			Item:    &model.Item{ID: id, Title: "(not found)", Status: "unknown"},
			Project: "?",
			Depth:   depth,
		}
	}

	node := &treeNode{
		Item:    info.item,
		Project: info.project,
		Depth:   depth,
	}

	if depth >= maxDepth {
		return node
	}

	seen := make(map[string]bool)
	for _, childID := range childrenOf[id] {
		if seen[childID] {
			continue
		}
		seen[childID] = true
		child := buildTreeNode(childID, allItems, childrenOf, depth+1, maxDepth)
		node.Children = append(node.Children, child)
	}

	return node
}

func renderTreeText(node *treeNode, prefix string, isLast bool) {
	connector := "├── "
	if isLast {
		connector = "└── "
	}
	if node.Depth == 0 {
		connector = ""
		prefix = ""
	}

	statusTag := node.Item.Status
	if node.Item.Blocked != nil {
		statusTag = "blocked"
	}

	fmt.Fprintf(os.Stdout, "%s%s%s [%-8s] [%s] %s\n",
		prefix, connector, node.Item.ID, statusTag, node.Item.Priority, node.Item.Title)

	childPrefix := prefix
	if node.Depth > 0 {
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}
	}

	for i, child := range node.Children {
		isLastChild := i == len(node.Children)-1
		renderTreeText(child, childPrefix, isLastChild)
	}
}

func renderTreeJSON(node *treeNode) {
	type jsonNode struct {
		ID       string      `json:"id"`
		Type     string      `json:"type"`
		Title    string      `json:"title"`
		Status   string      `json:"status"`
		Priority string      `json:"priority"`
		Phase    *int        `json:"phase,omitempty"`
		Project  string      `json:"project"`
		Children []*jsonNode `json:"children,omitempty"`
	}

	var convert func(n *treeNode) *jsonNode
	convert = func(n *treeNode) *jsonNode {
		jn := &jsonNode{
			ID:       n.Item.ID,
			Type:     string(n.Item.Type),
			Title:    n.Item.Title,
			Status:   n.Item.Status,
			Priority: n.Item.Priority,
			Phase:    n.Item.Phase,
			Project:  n.Project,
		}
		for _, child := range n.Children {
			jn.Children = append(jn.Children, convert(child))
		}
		return jn
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(convert(node))
}

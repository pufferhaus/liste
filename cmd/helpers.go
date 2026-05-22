package cmd

import (
	"github.com/pufferhaus/liste/internal/model"
	"github.com/pufferhaus/liste/internal/store"
)

// itemWithProject pairs an item with its owning project name.
// Used across multiple commands that aggregate items from all projects.
type itemWithProject struct {
	item    *model.Item
	project string
}

// priorityWeight returns a numeric weight for sorting (lower = higher priority).
func priorityWeight(p string) int {
	switch p {
	case "critical":
		return 0
	case "high":
		return 1
	case "medium":
		return 2
	case "low":
		return 3
	default:
		return 4
	}
}

// phaseOrder returns a sort key for phase (lower = earlier, unphased items sort last).
func phaseOrder(item *model.Item) int {
	if item.Phase == nil {
		return 9999
	}
	return *item.Phase
}

// depsResolved checks if all depends-on links point to done/cancelled items.
func depsResolved(item *model.Item, allItems map[string]*model.Item) bool {
	for _, link := range item.Links {
		if link.Type != model.LinkDependsOn {
			continue
		}
		target, ok := allItems[link.Target]
		if !ok {
			return false
		}
		if target.Status != "done" && target.Status != "cancelled" {
			return false
		}
	}
	return true
}

// collectAllItems gathers items from root and all sub-projects.
func collectAllItems() ([]itemWithProject, map[string]*model.Item, string, error) {
	result, err := getDiscovery()
	if err != nil {
		return nil, nil, "", err
	}

	var allItems []itemWithProject
	allItemsByID := make(map[string]*model.Item)

	rootStore := store.New(result.Root)
	rootCfg, err := rootStore.ReadConfig()
	if err != nil {
		return nil, nil, "", err
	}
	rootItems, err := rootStore.ListItems()
	if err != nil {
		return nil, nil, "", err
	}
	for _, item := range rootItems {
		allItems = append(allItems, itemWithProject{item: item, project: rootCfg.Project})
		allItemsByID[item.ID] = item
	}

	for _, sub := range result.SubProjects {
		s := store.New(sub.Path)
		items, err := s.ListItems()
		if err != nil {
			continue
		}
		for _, item := range items {
			allItems = append(allItems, itemWithProject{item: item, project: sub.Name})
			allItemsByID[item.ID] = item
		}
	}

	return allItems, allItemsByID, rootCfg.Project, nil
}

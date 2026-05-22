package resolver

import (
	"github.com/pufferhaus/liste/internal/model"
	"github.com/pufferhaus/liste/internal/store"
)

// InverseLink represents a link from another item pointing at a given item.
type InverseLink struct {
	Type     model.LinkType
	SourceID string
	Project  string // empty if same project
}

// ResolveInverse finds all items that link TO the given item ID.
// This enables bidirectional display without redundant storage.
func ResolveInverse(s *store.Store, targetID string) ([]InverseLink, error) {
	items, err := s.ListItems()
	if err != nil {
		return nil, err
	}

	var results []InverseLink
	for _, item := range items {
		if item.ID == targetID {
			continue
		}
		for _, link := range item.Links {
			if link.Target == targetID && link.Project == "" {
				results = append(results, InverseLink{
					Type:     link.Type.Inverse(),
					SourceID: item.ID,
				})
			}
		}
	}

	return results, nil
}

// ResolveGraph builds the full link graph for an item (direct + inverse links).
type GraphNode struct {
	ID       string
	Title    string
	Status   string
	Relation model.LinkType
	Project  string
}

// BuildGraph returns all items connected to the given item.
func BuildGraph(s *store.Store, itemID string) ([]GraphNode, error) {
	item, err := s.ReadItem(itemID)
	if err != nil {
		return nil, err
	}

	var nodes []GraphNode

	// Direct links
	for _, link := range item.Links {
		target, err := s.ReadItem(link.Target)
		if err != nil {
			// Item might be in another project or deleted
			nodes = append(nodes, GraphNode{
				ID:       link.Target,
				Title:    "(unresolved)",
				Relation: link.Type,
				Project:  link.Project,
			})
			continue
		}
		nodes = append(nodes, GraphNode{
			ID:       target.ID,
			Title:    target.Title,
			Status:   target.Status,
			Relation: link.Type,
			Project:  link.Project,
		})
	}

	// Inverse links
	inverse, err := ResolveInverse(s, itemID)
	if err != nil {
		return nodes, nil // return what we have
	}

	for _, inv := range inverse {
		source, err := s.ReadItem(inv.SourceID)
		if err != nil {
			continue
		}
		nodes = append(nodes, GraphNode{
			ID:       source.ID,
			Title:    source.Title,
			Status:   source.Status,
			Relation: inv.Type,
			Project:  inv.Project,
		})
	}

	return nodes, nil
}

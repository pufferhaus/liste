package resolver

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pufferhaus/liste/internal/model"
	"github.com/pufferhaus/liste/internal/store"
)

func setupTestStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	roadmapPath := filepath.Join(dir, ".liste")
	s := store.New(roadmapPath)
	if err := s.Init("test"); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	return s
}

func writeTestItem(t *testing.T, s *store.Store, item *model.Item) {
	t.Helper()
	if err := s.WriteItem(item); err != nil {
		t.Fatalf("WriteItem failed: %v", err)
	}
}

func TestResolveInverse(t *testing.T) {
	s := setupTestStore(t)

	now := time.Now()
	writeTestItem(t, s, &model.Item{
		ID: "FEAT-001", Type: model.TypeFeature, Title: "Feature 1",
		Status: "active", Priority: "high", Created: now, Updated: now,
		Links: []model.Link{
			{Type: model.LinkDependsOn, Target: "TASK-001"},
		},
	})
	writeTestItem(t, s, &model.Item{
		ID: "TASK-001", Type: model.TypeTask, Title: "Task 1",
		Status: "planned", Priority: "medium", Created: now, Updated: now,
	})

	// TASK-001 should have inverse link "blocks FEAT-001"
	inverse, err := ResolveInverse(s, "TASK-001")
	if err != nil {
		t.Fatalf("ResolveInverse failed: %v", err)
	}

	if len(inverse) != 1 {
		t.Fatalf("Inverse links = %d, want 1", len(inverse))
	}
	if inverse[0].Type != model.LinkBlocks {
		t.Errorf("Inverse type = %q, want %q", inverse[0].Type, model.LinkBlocks)
	}
	if inverse[0].SourceID != "FEAT-001" {
		t.Errorf("Inverse source = %q, want FEAT-001", inverse[0].SourceID)
	}
}

func TestResolveInverseNoLinks(t *testing.T) {
	s := setupTestStore(t)

	now := time.Now()
	writeTestItem(t, s, &model.Item{
		ID: "FEAT-001", Type: model.TypeFeature, Title: "Feature 1",
		Status: "active", Priority: "high", Created: now, Updated: now,
	})

	inverse, err := ResolveInverse(s, "FEAT-001")
	if err != nil {
		t.Fatalf("ResolveInverse failed: %v", err)
	}
	if len(inverse) != 0 {
		t.Errorf("Inverse links = %d, want 0", len(inverse))
	}
}

func TestBuildGraph(t *testing.T) {
	s := setupTestStore(t)

	now := time.Now()
	writeTestItem(t, s, &model.Item{
		ID: "EPIC-001", Type: model.TypeEpic, Title: "Epic",
		Status: "idea", Priority: "medium", Created: now, Updated: now,
		Links: []model.Link{
			{Type: model.LinkParentOf, Target: "FEAT-001"},
		},
	})
	writeTestItem(t, s, &model.Item{
		ID: "FEAT-001", Type: model.TypeFeature, Title: "Feature",
		Status: "active", Priority: "high", Created: now, Updated: now,
		Links: []model.Link{
			{Type: model.LinkChildOf, Target: "EPIC-001"},
			{Type: model.LinkDependsOn, Target: "TASK-001"},
		},
	})
	writeTestItem(t, s, &model.Item{
		ID: "TASK-001", Type: model.TypeTask, Title: "Task",
		Status: "done", Priority: "medium", Created: now, Updated: now,
	})

	nodes, err := BuildGraph(s, "FEAT-001")
	if err != nil {
		t.Fatalf("BuildGraph failed: %v", err)
	}

	// Should have: child-of EPIC-001, depends-on TASK-001, and inverse parent-of from EPIC-001
	if len(nodes) < 2 {
		t.Errorf("Graph nodes = %d, want at least 2", len(nodes))
	}

	// Verify we can find the depends-on link
	found := false
	for _, n := range nodes {
		if n.ID == "TASK-001" && n.Relation == model.LinkDependsOn {
			found = true
			break
		}
	}
	if !found {
		t.Error("Missing depends-on TASK-001 in graph")
	}
}

func TestDepsResolved(t *testing.T) {
	dir := t.TempDir()
	roadmapPath := filepath.Join(dir, ".liste")
	if err := os.MkdirAll(roadmapPath, 0755); err != nil {
		t.Fatal(err)
	}
	s := store.New(roadmapPath)
	if err := s.Init("test"); err != nil {
		t.Fatal(err)
	}

	now := time.Now()

	// Target is done
	doneItem := &model.Item{
		ID: "TASK-001", Type: model.TypeTask, Title: "Done Task",
		Status: "done", Priority: "medium", Created: now, Updated: now,
	}
	if err := s.WriteItem(doneItem); err != nil {
		t.Fatal(err)
	}

	// Target is not done
	pendingItem := &model.Item{
		ID: "TASK-002", Type: model.TypeTask, Title: "Pending Task",
		Status: "planned", Priority: "medium", Created: now, Updated: now,
	}
	if err := s.WriteItem(pendingItem); err != nil {
		t.Fatal(err)
	}

	allItems := map[string]*model.Item{
		"TASK-001": doneItem,
		"TASK-002": pendingItem,
	}

	// Item with resolved dep
	resolved := &model.Item{
		Links: []model.Link{{Type: model.LinkDependsOn, Target: "TASK-001"}},
	}
	if !depsResolvedHelper(resolved, allItems) {
		t.Error("Expected deps resolved for done target")
	}

	// Item with unresolved dep
	unresolved := &model.Item{
		Links: []model.Link{{Type: model.LinkDependsOn, Target: "TASK-002"}},
	}
	if depsResolvedHelper(unresolved, allItems) {
		t.Error("Expected deps NOT resolved for pending target")
	}

	// Item with no deps
	noDeps := &model.Item{}
	if !depsResolvedHelper(noDeps, allItems) {
		t.Error("Expected deps resolved for item with no deps")
	}
}

// depsResolvedHelper replicates the logic from cmd package for testing.
func depsResolvedHelper(item *model.Item, allItems map[string]*model.Item) bool {
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

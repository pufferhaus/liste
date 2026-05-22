package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pufferhaus/liste/internal/model"
)

func mustInit(t *testing.T, s *Store, name string) {
	t.Helper()
	if err := s.Init(name); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
}

func TestStoreInitAndExists(t *testing.T) {
	dir := t.TempDir()
	roadmapPath := filepath.Join(dir, ".liste")
	s := New(roadmapPath)

	if s.Exists() {
		t.Error("Exists() = true before Init")
	}

	if err := s.Init("test-project"); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if !s.Exists() {
		t.Error("Exists() = false after Init")
	}

	// Verify config
	cfg, err := s.ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig failed: %v", err)
	}
	if cfg.Project != "test-project" {
		t.Errorf("Config.Project = %q, want %q", cfg.Project, "test-project")
	}
	if len(cfg.Statuses) == 0 {
		t.Error("Config.Statuses is empty")
	}

	// Verify state
	state, err := s.ReadState()
	if err != nil {
		t.Fatalf("ReadState failed: %v", err)
	}
	if state.NextIDs["FEAT"] != 1 {
		t.Errorf("State.NextIDs[FEAT] = %d, want 1", state.NextIDs["FEAT"])
	}
}

func TestStoreCreateAndReadItem(t *testing.T) {
	dir := t.TempDir()
	roadmapPath := filepath.Join(dir, ".liste")
	s := New(roadmapPath)
	mustInit(t, s, "test")

	cfg, _ := s.ReadConfig()
	item, err := s.CreateItem(model.TypeFeature, "Test Feature", cfg)
	if err != nil {
		t.Fatalf("CreateItem failed: %v", err)
	}

	if item.ID != "FEAT-001" {
		t.Errorf("ID = %q, want %q", item.ID, "FEAT-001")
	}
	if item.Status != "idea" {
		t.Errorf("Status = %q, want %q", item.Status, "idea")
	}

	// Read it back
	read, err := s.ReadItem("FEAT-001")
	if err != nil {
		t.Fatalf("ReadItem failed: %v", err)
	}
	if read.Title != "Test Feature" {
		t.Errorf("Title = %q, want %q", read.Title, "Test Feature")
	}
}

func TestStoreNextIDIncrement(t *testing.T) {
	dir := t.TempDir()
	roadmapPath := filepath.Join(dir, ".liste")
	s := New(roadmapPath)
	mustInit(t, s, "test")

	cfg, _ := s.ReadConfig()

	item1, _ := s.CreateItem(model.TypeBug, "Bug 1", cfg)
	item2, _ := s.CreateItem(model.TypeBug, "Bug 2", cfg)
	item3, _ := s.CreateItem(model.TypeFeature, "Feature 1", cfg)

	if item1.ID != "BUG-001" {
		t.Errorf("item1.ID = %q, want BUG-001", item1.ID)
	}
	if item2.ID != "BUG-002" {
		t.Errorf("item2.ID = %q, want BUG-002", item2.ID)
	}
	if item3.ID != "FEAT-001" {
		t.Errorf("item3.ID = %q, want FEAT-001", item3.ID)
	}
}

func TestStoreListItems(t *testing.T) {
	dir := t.TempDir()
	roadmapPath := filepath.Join(dir, ".liste")
	s := New(roadmapPath)
	mustInit(t, s, "test")

	cfg, _ := s.ReadConfig()
	if _, err := s.CreateItem(model.TypeFeature, "Feature A", cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateItem(model.TypeBug, "Bug B", cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateItem(model.TypeTask, "Task C", cfg); err != nil {
		t.Fatal(err)
	}

	items, err := s.ListItems()
	if err != nil {
		t.Fatalf("ListItems failed: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("ListItems returned %d items, want 3", len(items))
	}
}

func TestStoreDeleteItem(t *testing.T) {
	dir := t.TempDir()
	roadmapPath := filepath.Join(dir, ".liste")
	s := New(roadmapPath)
	mustInit(t, s, "test")

	cfg, _ := s.ReadConfig()
	if _, err := s.CreateItem(model.TypeFeature, "To Delete", cfg); err != nil {
		t.Fatal(err)
	}

	if err := s.DeleteItem("FEAT-001"); err != nil {
		t.Fatalf("DeleteItem failed: %v", err)
	}

	_, err := s.ReadItem("FEAT-001")
	if err == nil {
		t.Error("ReadItem after delete should fail")
	}
}

func TestStoreDeleteItemNotFound(t *testing.T) {
	dir := t.TempDir()
	roadmapPath := filepath.Join(dir, ".liste")
	s := New(roadmapPath)
	mustInit(t, s, "test")

	err := s.DeleteItem("NONEXISTENT-001")
	if err == nil {
		t.Error("DeleteItem for non-existent item should fail")
	}
}

func TestStoreIgnoresNonItemFiles(t *testing.T) {
	dir := t.TempDir()
	roadmapPath := filepath.Join(dir, ".liste")
	s := New(roadmapPath)
	mustInit(t, s, "test")

	// Create a non-item markdown file
	if err := os.WriteFile(filepath.Join(roadmapPath, "notes.md"), []byte("# Notes\nJust some notes."), 0644); err != nil {
		t.Fatal(err)
	}

	items, err := s.ListItems()
	if err != nil {
		t.Fatalf("ListItems failed: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("ListItems returned %d items, want 0 (should skip notes.md)", len(items))
	}
}

func TestStoreInitAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	roadmapPath := filepath.Join(dir, ".liste")
	s := New(roadmapPath)
	mustInit(t, s, "test")

	// Second init should still work (idempotent at store level)
	// The CLI prevents this, but the store layer just overwrites
	if err := s.Init("test2"); err != nil {
		t.Fatalf("Second Init failed: %v", err)
	}
}

func TestStoreCorruptedState(t *testing.T) {
	dir := t.TempDir()
	roadmapPath := filepath.Join(dir, ".liste")
	s := New(roadmapPath)
	mustInit(t, s, "test")

	// Corrupt the state file
	if err := os.WriteFile(filepath.Join(roadmapPath, ".state.yaml"), []byte("{{invalid yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	// Should recover gracefully
	state, err := s.ReadState()
	if err != nil {
		t.Fatalf("ReadState with corrupted file should recover, got: %v", err)
	}
	if state.NextIDs["FEAT"] != 1 {
		t.Errorf("Recovered state should have default FEAT=1, got %d", state.NextIDs["FEAT"])
	}
}

func TestStoreEmptyState(t *testing.T) {
	dir := t.TempDir()
	roadmapPath := filepath.Join(dir, ".liste")
	s := New(roadmapPath)
	mustInit(t, s, "test")

	// Empty the state file
	if err := os.WriteFile(filepath.Join(roadmapPath, ".state.yaml"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	state, err := s.ReadState()
	if err != nil {
		t.Fatalf("ReadState with empty file should recover, got: %v", err)
	}
	if state.NextIDs == nil {
		t.Error("Recovered state should have NextIDs initialized")
	}
}

func TestStoreConfigMissingProject(t *testing.T) {
	dir := t.TempDir()
	roadmapPath := filepath.Join(dir, ".liste")
	if err := os.MkdirAll(roadmapPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Write a config without the project field
	if err := os.WriteFile(filepath.Join(roadmapPath, "config.yaml"), []byte("statuses: [idea, done]\n"), 0644); err != nil {
		t.Fatal(err)
	}

	s := New(roadmapPath)
	_, err := s.ReadConfig()
	if err == nil {
		t.Error("ReadConfig with missing project should fail")
	}
}

func TestStoreWriteNilItem(t *testing.T) {
	dir := t.TempDir()
	roadmapPath := filepath.Join(dir, ".liste")
	s := New(roadmapPath)
	mustInit(t, s, "test")

	err := s.WriteItem(nil)
	if err == nil {
		t.Error("WriteItem(nil) should fail")
	}
}

func TestStoreWriteItemEmptyID(t *testing.T) {
	dir := t.TempDir()
	roadmapPath := filepath.Join(dir, ".liste")
	s := New(roadmapPath)
	mustInit(t, s, "test")

	err := s.WriteItem(&model.Item{Title: "no id"})
	if err == nil {
		t.Error("WriteItem with empty ID should fail")
	}
}

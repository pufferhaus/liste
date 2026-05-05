package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
}

func TestFindRootFromExactDir(t *testing.T) {
	dir := t.TempDir()
	roadmapDir := filepath.Join(dir, ".liste")
	if err := os.MkdirAll(roadmapDir, 0755); err != nil {
		t.Fatal(err)
	}

	result, err := FindRoot(dir)
	if err != nil {
		t.Fatalf("FindRoot failed: %v", err)
	}
	if result != roadmapDir {
		t.Errorf("FindRoot = %q, want %q", result, roadmapDir)
	}
}

func TestFindRootFromSubdir(t *testing.T) {
	dir := t.TempDir()
	roadmapDir := filepath.Join(dir, ".liste")
	if err := os.MkdirAll(roadmapDir, 0755); err != nil {
		t.Fatal(err)
	}

	subDir := filepath.Join(dir, "sub", "deep")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	result, err := FindRoot(subDir)
	if err != nil {
		t.Fatalf("FindRoot failed: %v", err)
	}
	if result != roadmapDir {
		t.Errorf("FindRoot = %q, want %q", result, roadmapDir)
	}
}

func TestFindRootNotFound(t *testing.T) {
	dir := t.TempDir()

	result, err := FindRoot(dir)
	if err != nil {
		t.Fatalf("FindRoot failed: %v", err)
	}
	if result != "" {
		t.Errorf("FindRoot = %q, want empty string", result)
	}
}

func TestFindSubProjects(t *testing.T) {
	dir := t.TempDir()

	// Create root .liste
	mustMkdir(t, filepath.Join(dir, ".liste"))

	// Create sub-project .liste directories
	mustMkdir(t, filepath.Join(dir, "service-a", ".liste"))
	mustMkdir(t, filepath.Join(dir, "service-b", ".liste"))
	mustMkdir(t, filepath.Join(dir, "nested", "service-c", ".liste"))

	subs, err := FindSubProjects(dir)
	if err != nil {
		t.Fatalf("FindSubProjects failed: %v", err)
	}

	if len(subs) != 3 {
		t.Fatalf("Found %d sub-projects, want 3", len(subs))
	}

	names := make(map[string]bool)
	for _, sub := range subs {
		names[sub.Name] = true
	}

	if !names["service-a"] {
		t.Error("Missing sub-project: service-a")
	}
	if !names["service-b"] {
		t.Error("Missing sub-project: service-b")
	}
	if !names["nested/service-c"] {
		t.Error("Missing sub-project: nested/service-c")
	}
}

func TestFindSubProjectsSkipsHiddenDirs(t *testing.T) {
	dir := t.TempDir()

	mustMkdir(t, filepath.Join(dir, ".liste"))
	mustMkdir(t, filepath.Join(dir, ".hidden", ".liste"))

	subs, err := FindSubProjects(dir)
	if err != nil {
		t.Fatalf("FindSubProjects failed: %v", err)
	}

	if len(subs) != 0 {
		t.Errorf("Found %d sub-projects, want 0 (should skip .hidden)", len(subs))
	}
}

func TestDiscoverFull(t *testing.T) {
	dir := t.TempDir()

	mustMkdir(t, filepath.Join(dir, ".liste"))
	mustMkdir(t, filepath.Join(dir, "svc", ".liste"))

	result, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if result == nil {
		t.Fatal("Discover returned nil")
	}
	if result.Root != filepath.Join(dir, ".liste") {
		t.Errorf("Root = %q, want %q", result.Root, filepath.Join(dir, ".liste"))
	}
	if len(result.SubProjects) != 1 {
		t.Errorf("SubProjects = %d, want 1", len(result.SubProjects))
	}
}

func TestDiscoverNone(t *testing.T) {
	dir := t.TempDir()

	result, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if result != nil {
		t.Errorf("Discover = %+v, want nil", result)
	}
}

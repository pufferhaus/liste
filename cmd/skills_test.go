package cmd

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"testing"

	listeskills "github.com/pufferhaus/liste/skills"
)

func TestSkillsEmbed(t *testing.T) {
	expected := []string{
		"session-start", "add-bug", "add-feature", "add-task", "add-idea",
		"add-epic", "start", "done", "block", "promote", "link", "find",
		"append", "set", "status", "next", "progress", "diff", "batch",
	}
	for _, name := range expected {
		path := path.Join(name, "SKILL.md")
		data, err := listeskills.Files.ReadFile(path)
		if err != nil {
			t.Errorf("skill %q not found in embed: %v", name, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("skill %q is empty", name)
		}
	}
}

func TestSkillsInstallCreatesFiles(t *testing.T) {
	tmp := t.TempDir()
	pluginsJSON := filepath.Join(tmp, "plugins", "installed_plugins.json")

	if err := os.MkdirAll(filepath.Dir(pluginsJSON), 0755); err != nil {
		t.Fatal(err)
	}
	initial := map[string]any{"version": 2, "plugins": map[string]any{}}
	data, _ := json.Marshal(initial)
	if err := os.WriteFile(pluginsJSON, data, 0644); err != nil {
		t.Fatal(err)
	}

	if err := installSkills(tmp, "test-version"); err != nil {
		t.Fatalf("installSkills: %v", err)
	}

	installDir := filepath.Join(tmp, "plugins", "cache", "liste", "liste", "test-version")
	expected := []string{
		"session-start", "add-bug", "add-feature", "add-task", "add-idea",
		"add-epic", "start", "done", "block", "promote", "link", "find",
		"append", "set", "status", "next", "progress", "diff", "batch",
	}
	for _, name := range expected {
		p := filepath.Join(installDir, "skills", name, "SKILL.md")
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected skill file missing: %s", p)
		}
	}

	pluginJSON := filepath.Join(installDir, ".claude-plugin", "plugin.json")
	if _, err := os.Stat(pluginJSON); err != nil {
		t.Errorf("plugin.json not written: %v", err)
	}

	raw, err := os.ReadFile(pluginsJSON)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	plugins, _ := result["plugins"].(map[string]any)
	if _, ok := plugins["liste@liste"]; !ok {
		t.Error("liste@liste not registered in installed_plugins.json")
	}
}

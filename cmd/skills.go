package cmd

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	listeskills "github.com/pblca/liste/skills"
	"github.com/spf13/cobra"
)

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Manage liste Claude Code skills",
	Long:  "Commands for installing and managing liste's Claude Code skills.",
}

var skillsInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install liste skills to ~/.claude",
	Long:  "Copies all liste Claude Code skills into ~/.claude/plugins/cache/liste/ and registers the plugin.",
	Args:  cobra.NoArgs,
	RunE:  runSkillsInstall,
}

var skillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available skills",
	Args:  cobra.NoArgs,
	RunE:  runSkillsList,
}

func init() {
	skillsCmd.AddCommand(skillsInstallCmd)
	skillsCmd.AddCommand(skillsListCmd)
	rootCmd.AddCommand(skillsCmd)
}

func runSkillsInstall(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("finding home directory: %w", err)
	}
	if err := installSkills(filepath.Join(home, ".claude"), buildVersion); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "\nAdd to your project's .claude/CLAUDE.md or ~/.claude/CLAUDE.md:\n\n")
	fmt.Fprintf(os.Stdout, "  At the start of every session, invoke the liste:session-start skill.\n")
	return nil
}

// installSkills copies embedded skill files and plugin manifest to claudeRoot
// and registers the plugin in installed_plugins.json.
func installSkills(claudeRoot, version string) error {
	installDir := filepath.Join(claudeRoot, "plugins", "cache", "liste", "liste", version)

	count := 0
	err := fs.WalkDir(listeskills.Files, listeskills.SkillsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := listeskills.Files.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading embedded file %s: %w", path, err)
		}
		// Install to installDir/skills/<name>/SKILL.md
		dst := filepath.Join(installDir, "skills", path)
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", path, err)
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", dst, err)
		}
		fmt.Fprintf(os.Stdout, "  ✓ %s\n", path)
		count++
		return nil
	})
	if err != nil {
		return err
	}

	pluginDir := filepath.Join(installDir, ".claude-plugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return fmt.Errorf("creating .claude-plugin dir: %w", err)
	}
	pluginJSON := `{
  "name": "liste",
  "description": "Claude Code skills for the liste CLI roadmap tracker.",
  "author": {
    "name": "pblca",
    "url": "https://github.com/pblca/liste"
  }
}`
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(pluginJSON), 0644); err != nil {
		return fmt.Errorf("writing plugin.json: %w", err)
	}

	if err := registerPlugin(claudeRoot, installDir, version); err != nil {
		return fmt.Errorf("updating installed_plugins.json: %w", err)
	}

	fmt.Fprintf(os.Stdout, "\nInstalled %d skill files to %s\n", count, installDir)
	return nil
}

func registerPlugin(claudeRoot, installPath, version string) error {
	pluginsFile := filepath.Join(claudeRoot, "plugins", "installed_plugins.json")

	var registry map[string]any
	data, err := os.ReadFile(pluginsFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		registry = map[string]any{"version": 2, "plugins": map[string]any{}}
	} else {
		if err := json.Unmarshal(data, &registry); err != nil {
			return fmt.Errorf("parsing installed_plugins.json: %w", err)
		}
	}

	plugins, _ := registry["plugins"].(map[string]any)
	if plugins == nil {
		plugins = map[string]any{}
		registry["plugins"] = plugins
	}

	now := time.Now().UTC().Format(time.RFC3339)
	plugins["liste@liste"] = []any{
		map[string]any{
			"scope":       "user",
			"installPath": installPath,
			"version":     version,
			"installedAt": now,
			"lastUpdated": now,
		},
	}

	out, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(pluginsFile), 0755); err != nil {
		return err
	}
	return os.WriteFile(pluginsFile, append(out, '\n'), 0644)
}

func runSkillsList(cmd *cobra.Command, args []string) error {
	entries, err := fs.ReadDir(listeskills.Files, listeskills.SkillsDir)
	if err != nil {
		return fmt.Errorf("reading skills: %w", err)
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			count++
		}
	}
	if flagQuiet {
		for _, e := range entries {
			if e.IsDir() {
				fmt.Fprintln(os.Stdout, "liste:"+e.Name())
			}
		}
		return nil
	}
	fmt.Fprintf(os.Stdout, "%d skills available (run 'liste skills install' to install):\n\n", count)
	for _, e := range entries {
		if e.IsDir() {
			fmt.Fprintf(os.Stdout, "  liste:%-20s  /liste-%s\n", e.Name(), e.Name())
		}
	}
	return nil
}

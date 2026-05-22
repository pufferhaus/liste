# liste Skills Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship 19 Claude Code skills with the `liste` binary so agents automatically use `liste` commands during development, and users can install them with `liste skills install`.

**Architecture:** Skill files live in `skills/<name>/SKILL.md` at the repo root, embedded into the binary via `internal/skills/embed.go`. The `liste skills install` command copies them into `~/.claude/plugins/cache/liste/liste/<version>/` and registers the plugin in `~/.claude/plugins/installed_plugins.json`. A `.claude-plugin/plugin.json` makes `liste` a first-class Claude Code plugin installable via `claude plugin install`.

**Tech Stack:** Go 1.26, `embed` package, Claude Code plugin system (SKILL.md format)

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `.claude-plugin/plugin.json` | Create | Plugin manifest for `claude plugin install` |
| `.claude-plugin/marketplace.json` | Create | Marketplace registration metadata |
| `skills/session-start/SKILL.md` | Create | Auto-trigger: context load + hard behavioral rules |
| `skills/add-bug/SKILL.md` | Create | Add a bug item |
| `skills/add-feature/SKILL.md` | Create | Add a feature item |
| `skills/add-task/SKILL.md` | Create | Add a task item |
| `skills/add-idea/SKILL.md` | Create | Add an idea item |
| `skills/add-epic/SKILL.md` | Create | Add an epic item |
| `skills/start/SKILL.md` | Create | Mark item active |
| `skills/done/SKILL.md` | Create | Mark item done |
| `skills/block/SKILL.md` | Create | Mark item blocked |
| `skills/promote/SKILL.md` | Create | Advance item to next status |
| `skills/link/SKILL.md` | Create | Create typed relationship between items |
| `skills/find/SKILL.md` | Create | Search before creating (dedup check) |
| `skills/append/SKILL.md` | Create | Add notes/findings to item |
| `skills/set/SKILL.md` | Create | Update item metadata |
| `skills/status/SKILL.md` | Create | Roadmap overview dashboard |
| `skills/next/SKILL.md` | Create | Show what to work on next |
| `skills/progress/SKILL.md` | Create | Completion progress view |
| `skills/diff/SKILL.md` | Create | Show changes since last check |
| `skills/batch/SKILL.md` | Create | Bulk mutations via stdin |
| `internal/skills/embed.go` | Create | `//go:embed` FS for all skill files |
| `cmd/skills.go` | Create | `liste skills` + `liste skills install` commands |
| `cmd/skills_test.go` | Create | Tests for install command |

---

## Task 1: Plugin manifest files

**Files:**
- Create: `.claude-plugin/plugin.json`
- Create: `.claude-plugin/marketplace.json`

- [ ] **Step 1: Create `.claude-plugin/plugin.json`**

```bash
mkdir -p .claude-plugin
```

Write `.claude-plugin/plugin.json`:

```json
{
  "name": "liste",
  "description": "Claude Code skills for the liste CLI roadmap tracker. Teaches agents to record bugs, track progress, manage dependencies, and keep roadmaps current during development.",
  "author": {
    "name": "pufferhaus",
    "url": "https://github.com/pufferhaus/liste"
  }
}
```

- [ ] **Step 2: Create `.claude-plugin/marketplace.json`**

Write `.claude-plugin/marketplace.json`:

```json
{
  "$schema": "https://anthropic.com/claude-code/marketplace.schema.json",
  "name": "liste",
  "description": "Claude Code skills for the liste CLI roadmap tracker",
  "owner": {
    "name": "pufferhaus",
    "url": "https://github.com/pufferhaus/liste"
  },
  "plugins": [
    {
      "name": "liste",
      "description": "Skills for managing projects with the liste CLI",
      "source": "./",
      "category": "productivity"
    }
  ]
}
```

- [ ] **Step 3: Verify build still passes**

```bash
export PATH="/opt/homebrew/bin:$PATH" && go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add .claude-plugin/
git commit -m "chore: add Claude Code plugin manifest files"
```

---

## Task 2: session-start skill

**Files:**
- Create: `skills/session-start/SKILL.md`

- [ ] **Step 1: Create the skill file**

```bash
mkdir -p skills/session-start
```

Write `skills/session-start/SKILL.md`:

```markdown
---
name: liste-session-start
description: >
  Load liste project context and install hard behavioral rules at session start.
  Invoke at the start of every session when working in a project directory.
  Triggers when CLAUDE.md says "invoke liste:session-start at session start" or similar.
---

# liste Session Start

## Step 1: Detect project

Run:
```bash
liste context 2>/dev/null
```

If the command fails with "no .liste/ found", this skill is a no-op — stop here.

If it succeeds, present the full context output to the user.

## Step 2: Hard behavioral rules

**These rules are mandatory for the entire session. No exceptions. No skipping.**

### Before creating any item
```bash
liste search "<keywords>"
```
Check results. If a matching item exists, update it with `liste append <id>` instead of creating a duplicate. Only proceed with creation if no match is found.

### Bug found
```bash
liste add bug "<concise description of the defect>"
```
Run this **immediately** when any bug, error, crash, or unexpected behavior is discovered — before continuing any other work.

### Starting work on an item
```bash
liste set <id> status active
```
Run this **before** beginning any work on a liste item.

### Completing an item
```bash
liste done <id>
```
Run this **immediately** when work on a liste item is complete.

### Blocked on an item
```bash
liste block <id> "<specific reason for the block>"
```
Run this when you cannot proceed with a liste item.

### Dependency discovered
```bash
liste link <id> depends-on <target-id>
```
Run this when you discover that one item cannot proceed until another is done.

### Significant finding about an item
```bash
liste append <id> "<what you learned>"
```
Run this when you discover important context, a constraint, or implementation detail about an item.

### Three or more mutations needed
```bash
liste batch <<EOF
done FEAT-001
set BUG-002 status active
add task "Write integration tests"
add bug "Login fails on Safari"
EOF
```
Use `liste batch` for 3+ operations. Never loop individual commands when batch is available.

## Step 3: Show work queue

```bash
liste next
```

Present the result to the user and ask if they'd like to start on the top item.
```

- [ ] **Step 2: Commit**

```bash
git add skills/session-start/
git commit -m "feat: add liste session-start skill with hard behavioral rules"
```

---

## Task 3: Creation skills (add-bug, add-feature, add-task, add-idea, add-epic)

**Files:**
- Create: `skills/add-bug/SKILL.md`
- Create: `skills/add-feature/SKILL.md`
- Create: `skills/add-task/SKILL.md`
- Create: `skills/add-idea/SKILL.md`
- Create: `skills/add-epic/SKILL.md`

- [ ] **Step 1: Create add-bug**

```bash
mkdir -p skills/add-bug
```

Write `skills/add-bug/SKILL.md`:

```markdown
---
name: liste-add-bug
description: >
  Add a bug to the liste roadmap. Use when a defect, error, crash, or unexpected
  behavior is discovered during development. Invoke as /liste-add-bug.
---

# Add Bug to liste

## Step 1: Search for duplicates

```bash
liste search "<keywords from the bug>"
```

If a matching bug exists: run `liste append <existing-id> "<additional context>"` and stop.

## Step 2: Add the bug

```bash
liste add bug "<concise title: what is wrong>"
```

## Step 3: Set priority

```bash
liste set <new-id> priority <level>
```

- `critical` — data loss, security, complete breakage, blocks all work
- `high` — significant feature broken, no workaround
- `medium` — partial breakage, workaround exists
- `low` — cosmetic, minor

## Step 4: Link if related

```bash
liste link <bug-id> blocks <item-id>      # if it blocks another item
liste link <bug-id> relates-to <item-id>  # if generally related
```
```

- [ ] **Step 2: Create add-feature**

```bash
mkdir -p skills/add-feature
```

Write `skills/add-feature/SKILL.md`:

```markdown
---
name: liste-add-feature
description: >
  Add a new feature to the liste roadmap. Use for new capabilities that don't yet
  exist. Invoke as /liste-add-feature.
---

# Add Feature to liste

## Step 1: Search for duplicates

```bash
liste search "<keywords from the feature>"
```

If a matching item exists: update it with `liste append <id>` and stop.

## Step 2: Add the feature

```bash
liste add feature "<concise title: what the feature does>"
```

## Step 3: Set metadata

```bash
liste set <new-id> priority <critical|high|medium|low>
liste set <new-id> phase <number>   # if known
```

## Step 4: Link dependencies

```bash
liste link <feature-id> depends-on <blocking-id>
```
```

- [ ] **Step 3: Create add-task**

```bash
mkdir -p skills/add-task
```

Write `skills/add-task/SKILL.md`:

```markdown
---
name: liste-add-task
description: >
  Add a concrete work unit to the liste roadmap. Use for specific, actionable
  items (write tests, update docs, refactor X). Invoke as /liste-add-task.
---

# Add Task to liste

## Step 1: Search for duplicates

```bash
liste search "<keywords from the task>"
```

If a matching item exists: update it with `liste append <id>` and stop.

## Step 2: Add the task

```bash
liste add task "<concise title: specific action to take>"
```

## Step 3: Set metadata

```bash
liste set <new-id> priority <critical|high|medium|low>
liste set <new-id> phase <number>
```

## Step 4: Link to parent

```bash
liste link <task-id> child-of <feature-or-epic-id>
```
```

- [ ] **Step 4: Create add-idea**

```bash
mkdir -p skills/add-idea
```

Write `skills/add-idea/SKILL.md`:

```markdown
---
name: liste-add-idea
description: >
  Add an unplanned concept or future possibility to the liste roadmap.
  Use for things worth capturing but not yet committed to.
  Invoke as /liste-add-idea.
---

# Add Idea to liste

## Step 1: Search for duplicates

```bash
liste search "<keywords from the idea>"
```

If a matching item exists: run `liste append <id> "<additional thoughts>"` and stop.

## Step 2: Add the idea

```bash
liste add idea "<concise title: what the idea is>"
```

Ideas start with status `idea` by default. No further metadata required.

## Optional: Add context

```bash
liste append <new-id> "<why this is interesting, constraints to keep in mind>"
```
```

- [ ] **Step 5: Create add-epic**

```bash
mkdir -p skills/add-epic
```

Write `skills/add-epic/SKILL.md`:

```markdown
---
name: liste-add-epic
description: >
  Add a large grouping of related work to the liste roadmap. Use for major
  initiatives containing multiple features, tasks, and bugs.
  Invoke as /liste-add-epic.
---

# Add Epic to liste

## Step 1: Search for duplicates

```bash
liste search "<keywords from the epic>"
```

## Step 2: Add the epic

```bash
liste add epic "<concise title: the initiative name>"
```

## Step 3: Assign phase and link children

```bash
liste set <new-id> phase <number>
liste link <child-id> child-of <epic-id>   # repeat for each child item
```
```

- [ ] **Step 6: Commit**

```bash
git add skills/add-bug/ skills/add-feature/ skills/add-task/ skills/add-idea/ skills/add-epic/
git commit -m "feat: add liste creation skills (add-bug, add-feature, add-task, add-idea, add-epic)"
```

---

## Task 4: Transition skills (start, done, block, promote)

**Files:**
- Create: `skills/start/SKILL.md`
- Create: `skills/done/SKILL.md`
- Create: `skills/block/SKILL.md`
- Create: `skills/promote/SKILL.md`

- [ ] **Step 1: Create start**

```bash
mkdir -p skills/start
```

Write `skills/start/SKILL.md`:

```markdown
---
name: liste-start
description: >
  Mark a liste item as active — you are beginning work on it now.
  Invoke as /liste-start or when beginning work on a planned item.
---

# Start a liste Item

```bash
liste set <id> status active
```

If you don't know the ID:
```bash
liste next                      # top priority items ready to start
liste list --status planned
liste search "<keywords>"
```
```

- [ ] **Step 2: Create done**

```bash
mkdir -p skills/done
```

Write `skills/done/SKILL.md`:

```markdown
---
name: liste-done
description: >
  Mark a liste item as complete. Invoke as /liste-done or immediately when
  finishing work on a liste item.
---

# Mark liste Item Done

```bash
liste done <id>
```

If you don't know the ID:
```bash
liste list --status active
liste search "<keywords>"
```

After marking done, check for newly unblocked items:
```bash
liste ready
```
```

- [ ] **Step 3: Create block**

```bash
mkdir -p skills/block
```

Write `skills/block/SKILL.md`:

```markdown
---
name: liste-block
description: >
  Mark a liste item as blocked. Use when you cannot proceed due to an external
  dependency, decision, or missing resource. Invoke as /liste-block.
---

# Block a liste Item

```bash
liste block <id> "<specific reason for the block>"
```

The reason must describe exactly what prevents progress:
- `"Waiting on OAuth provider approval"`
- `"Depends on FEAT-003 which is not yet merged"`
- `"API contract not finalized with external team"`

To unblock later when resolved:
```bash
liste move <id> active
```
```

- [ ] **Step 4: Create promote**

```bash
mkdir -p skills/promote
```

Write `skills/promote/SKILL.md`:

```markdown
---
name: liste-promote
description: >
  Advance a liste item to its next status in the lifecycle.
  Default lifecycle: idea → planned → active → done.
  Invoke as /liste-promote.
---

# Promote a liste Item

```bash
liste promote <id>
```

To view current status first:
```bash
liste show <id>
```

To jump to a specific status instead of incrementing:
```bash
liste move <id> <status>
```
```

- [ ] **Step 5: Commit**

```bash
git add skills/start/ skills/done/ skills/block/ skills/promote/
git commit -m "feat: add liste transition skills (start, done, block, promote)"
```

---

## Task 5: Relationship and discovery skills (link, find)

**Files:**
- Create: `skills/link/SKILL.md`
- Create: `skills/find/SKILL.md`

- [ ] **Step 1: Create link**

```bash
mkdir -p skills/link
```

Write `skills/link/SKILL.md`:

```markdown
---
name: liste-link
description: >
  Create a typed relationship between two liste items.
  Use when you discover a dependency, hierarchy, or association.
  Invoke as /liste-link.
---

# Link liste Items

```bash
liste link <source-id> <type> <target-id>
```

## Link types

| Type | Meaning | Example |
|---|---|---|
| `depends-on` | Source cannot proceed until target is done | `liste link FEAT-001 depends-on TASK-003` |
| `blocks` | Source prevents target from proceeding | `liste link BUG-002 blocks FEAT-001` |
| `parent-of` | Source is a grouping containing target | `liste link EPIC-001 parent-of FEAT-002` |
| `child-of` | Source belongs to target grouping | `liste link FEAT-002 child-of EPIC-001` |
| `relates-to` | General association | `liste link FEAT-001 relates-to FEAT-004` |

## Remove a link

```bash
liste unlink <source-id> <type> <target-id>
```

## View full link graph

```bash
liste graph <id>
```
```

- [ ] **Step 2: Create find**

```bash
mkdir -p skills/find
```

Write `skills/find/SKILL.md`:

```markdown
---
name: liste-find
description: >
  Search for existing liste items before creating new ones. Always run before
  any add command to prevent duplicates. Invoke as /liste-find.
---

# Find in liste

Always search before creating:

```bash
liste search "<keywords>"
```

**Match found** → update instead of creating a duplicate:
```bash
liste append <id> "<additional context>"
liste set <id> priority <level>
liste set <id> phase <number>
```

**No match** → proceed with the appropriate creation skill.

## Filtered browsing

```bash
liste list --type bug
liste list --status active
liste list --priority critical
liste list --tag <tag>
```

## View hierarchy

```bash
liste tree
liste tree <id>
```
```

- [ ] **Step 3: Commit**

```bash
git add skills/link/ skills/find/
git commit -m "feat: add liste relationship and discovery skills (link, find)"
```

---

## Task 6: Annotation and metadata skills (append, set)

**Files:**
- Create: `skills/append/SKILL.md`
- Create: `skills/set/SKILL.md`

- [ ] **Step 1: Create append**

```bash
mkdir -p skills/append
```

Write `skills/append/SKILL.md`:

```markdown
---
name: liste-append
description: >
  Add notes, findings, or context to an existing liste item's body.
  Use when you learn something significant about an item during work.
  Invoke as /liste-append.
---

# Append to a liste Item

```bash
liste append <id> "<text to add>"
```

Use when you discover:
- An implementation constraint or gotcha
- A decision and its rationale
- Test results or benchmarks
- Links to relevant PRs, issues, or documentation
- Changes to scope or acceptance criteria

For longer edits:
```bash
liste edit <id>
```
```

- [ ] **Step 2: Create set**

```bash
mkdir -p skills/set
```

Write `skills/set/SKILL.md`:

```markdown
---
name: liste-set
description: >
  Update a field on an existing liste item. Use to change priority, phase,
  status, or tags. Invoke as /liste-set.
---

# Set a Field on a liste Item

```bash
liste set <id> <field> <value>
```

## Common fields

| Field | Values | Example |
|---|---|---|
| `priority` | critical, high, medium, low | `liste set FEAT-001 priority high` |
| `phase` | positive integer | `liste set FEAT-001 phase 2` |
| `status` | idea, planned, active, done, cancelled | `liste set FEAT-001 status planned` |

## Adding tags

```bash
liste set <id> tags backend,auth,security
```

## View current values

```bash
liste show <id>
```
```

- [ ] **Step 3: Commit**

```bash
git add skills/append/ skills/set/
git commit -m "feat: add liste annotation skills (append, set)"
```

---

## Task 7: Reporting skills (status, next, progress, diff, batch)

**Files:**
- Create: `skills/status/SKILL.md`
- Create: `skills/next/SKILL.md`
- Create: `skills/progress/SKILL.md`
- Create: `skills/diff/SKILL.md`
- Create: `skills/batch/SKILL.md`

- [ ] **Step 1: Create status**

```bash
mkdir -p skills/status
```

Write `skills/status/SKILL.md`:

```markdown
---
name: liste-status
description: >
  Show a full roadmap overview: phases, status counts, and blocked items.
  Use for a command-center view of current project state. Invoke as /liste-status.
---

# liste Status Overview

## Phase overview

```bash
liste roadmap
```

## Status counts

```bash
liste status
```

## Blocked items

```bash
liste blocked
```

## Compact AI-agent summary

```bash
liste context
```
```

- [ ] **Step 2: Create next**

```bash
mkdir -p skills/next
```

Write `skills/next/SKILL.md`:

```markdown
---
name: liste-next
description: >
  Show the highest-priority items ready to work on next.
  Invoke as /liste-next.
---

# What to Work on Next

## Single highest-priority item

```bash
liste next
```

## Multiple candidates

```bash
liste next --count 5
```

## Items with all dependencies satisfied

```bash
liste ready
```

## Items currently in progress

```bash
liste list --status active
```

To start an item from these results:
```bash
liste set <id> status active
```
```

- [ ] **Step 3: Create progress**

```bash
mkdir -p skills/progress
```

Write `skills/progress/SKILL.md`:

```markdown
---
name: liste-progress
description: >
  Show completion progress across the roadmap. Use to see done vs remaining.
  Invoke as /liste-progress.
---

# liste Completion Progress

```bash
liste progress
```

## By phase

```bash
liste phase <number>
```

## Items not updated recently

```bash
liste stale
```
```

- [ ] **Step 4: Create diff**

```bash
mkdir -p skills/diff
```

Write `skills/diff/SKILL.md`:

```markdown
---
name: liste-diff
description: >
  Show what has changed in the roadmap since the last check.
  Use at session start/end to see recent activity. Invoke as /liste-diff.
---

# liste Changes Since Last Check

```bash
liste diff
```

Shows items created, updated, or status-changed since the last time `liste diff`
was run. Useful for session hand-offs and progress summaries.
```

- [ ] **Step 5: Create batch**

```bash
mkdir -p skills/batch
```

Write `skills/batch/SKILL.md`:

```markdown
---
name: liste-batch
description: >
  Run multiple liste commands atomically via stdin. Use when making 3 or more
  mutations in a single operation. Invoke as /liste-batch.
---

# Batch liste Mutations

Use for 3 or more mutations — never loop individual commands:

```bash
liste batch <<EOF
done FEAT-001
set BUG-002 status active
add task "Write integration tests"
add bug "Login fails on Safari"
link TASK-003 depends-on FEAT-001
EOF
```

Commands in batch (without the `liste` prefix):
`add`, `done`, `set`, `move`, `block`, `link`, `unlink`, `delete`, `append`

## From file

```bash
liste batch < mutations.txt
```
```

- [ ] **Step 6: Commit**

```bash
git add skills/status/ skills/next/ skills/progress/ skills/diff/ skills/batch/
git commit -m "feat: add liste reporting and batch skills (status, next, progress, diff, batch)"
```

---

## Task 8: internal/skills embed package

**Files:**
- Create: `internal/skills/embed.go`

- [ ] **Step 1: Create the embed package**

```bash
mkdir -p internal/skills
```

Write `internal/skills/embed.go`:

```go
package skills

import "embed"

// Files contains all skill files embedded at compile time.
// Directory structure preserved: skills/<name>/SKILL.md
//
//go:embed ../../skills
var Files embed.FS

// SkillsDir is the root directory name within the embedded FS.
const SkillsDir = "skills"
```

- [ ] **Step 2: Verify it compiles**

```bash
export PATH="/opt/homebrew/bin:$PATH" && go build ./internal/skills/...
```
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/skills/embed.go
git commit -m "feat: add internal/skills embed package"
```

---

## Task 9: `liste skills install` command

**Files:**
- Create: `cmd/skills.go`
- Create: `cmd/skills_test.go`

- [ ] **Step 1: Write the failing test**

Create `cmd/skills_test.go`:

```go
package cmd_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pufferhaus/liste/internal/skills"
)

func TestSkillsEmbed(t *testing.T) {
	expected := []string{
		"session-start", "add-bug", "add-feature", "add-task", "add-idea",
		"add-epic", "start", "done", "block", "promote", "link", "find",
		"append", "set", "status", "next", "progress", "diff", "batch",
	}
	for _, name := range expected {
		path := filepath.Join(skills.SkillsDir, name, "SKILL.md")
		data, err := skills.Files.ReadFile(path)
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
```

- [ ] **Step 2: Run to confirm failure**

```bash
export PATH="/opt/homebrew/bin:$PATH" && go test ./cmd/... -run TestSkills 2>&1
```
Expected: FAIL — `installSkills` undefined.

- [ ] **Step 3: Create `cmd/skills.go`**

Write `cmd/skills.go`:

```go
package cmd

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/pufferhaus/liste/internal/skills"
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
	fmt.Fprintln(os.Stdout, "\nAdd to your project's .claude/CLAUDE.md or ~/.claude/CLAUDE.md:\n")
	fmt.Fprintln(os.Stdout, "  At the start of every session, invoke the liste:session-start skill.")
	return nil
}

// installSkills copies embedded skill files and plugin manifest to claudeRoot
// and registers the plugin in installed_plugins.json.
func installSkills(claudeRoot, version string) error {
	installDir := filepath.Join(claudeRoot, "plugins", "cache", "liste", "liste", version)

	count := 0
	err := fs.WalkDir(skills.Files, skills.SkillsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := skills.Files.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading embedded file %s: %w", path, err)
		}
		dst := filepath.Join(installDir, path)
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
    "name": "pufferhaus",
    "url": "https://github.com/pufferhaus/liste"
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
	entries, err := fs.ReadDir(skills.Files, skills.SkillsDir)
	if err != nil {
		return fmt.Errorf("reading skills: %w", err)
	}
	if flagQuiet {
		for _, e := range entries {
			if e.IsDir() {
				fmt.Fprintln(os.Stdout, "liste:"+e.Name())
			}
		}
		return nil
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			count++
		}
	}
	fmt.Fprintf(os.Stdout, "%d skills available (run 'liste skills install' to install):\n\n", count)
	for _, e := range entries {
		if e.IsDir() {
			fmt.Fprintf(os.Stdout, "  liste:%-20s  /liste-%s\n", e.Name(), e.Name())
		}
	}
	return nil
}
```

- [ ] **Step 4: Run tests**

```bash
export PATH="/opt/homebrew/bin:$PATH" && go test ./cmd/... -run TestSkills -v
```
Expected: PASS — both tests.

- [ ] **Step 5: Run full test suite**

```bash
export PATH="/opt/homebrew/bin:$PATH" && go test ./...
```
Expected: all pass.

- [ ] **Step 6: Smoke-test**

```bash
export PATH="/opt/homebrew/bin:$PATH" && go run . skills list
```
Expected: prints 19 skills with trigger names.

- [ ] **Step 7: Commit**

```bash
git add cmd/skills.go cmd/skills_test.go
git commit -m "feat: add liste skills install command with embedded skill files"
```

---

## Self-Review Notes

- **Spec coverage:** 19 skills ✅, hard behavioral rules in session-start ✅, `liste skills install` ✅, `liste skills list` ✅, plugin manifests ✅, `go:embed` ✅, `installed_plugins.json` updated ✅
- **Embed path:** `//go:embed ../../skills` in `internal/skills/embed.go` resolves to `liste/skills/` (two levels up from `internal/skills/`). Go's embed is relative to the source file's package directory — correct.
- **Type consistency:** `installSkills(claudeRoot, version string) error` defined in Task 9 Step 3, referenced in test (Step 1) and `runSkillsInstall`. Consistent.
- **Ordering:** Tasks 2–7 (skill files) must exist before Task 8 (embed compiles). Task 8 must complete before Task 9 (cmd/skills.go imports it). Task 1 is independent.

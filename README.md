# liste

```
  ██╗     ██╗███████╗████████╗███████╗
  ██║     ██║██╔════╝╚══██╔══╝██╔════╝
  ██║     ██║███████╗   ██║   █████╗  
  ██║     ██║╚════██║   ██║   ██╔══╝  
  ███████╗██║███████║   ██║   ███████╗
  ╚══════╝╚═╝╚══════╝   ╚═╝   ╚══════╝
  portable cli roadmap tracker · for humans and AI agents
```

A portable CLI tool for managing project roadmaps as structured markdown files. Designed for both humans and AI agents.

Items are stored as individual markdown files with YAML frontmatter in a `.liste/` directory. No database, no server, no lock-in — just files in your repo.

## Install

### Homebrew (macOS / Linux)

```bash
brew install pufferhaus/liste/liste
```

### Scoop (Windows)

```bash
scoop bucket add liste https://github.com/pufferhaus/liste
scoop install liste
```

### Go install

```bash
go install github.com/pufferhaus/liste@latest
```

### Release binaries

Download the latest binary for your platform from [Releases](https://github.com/pufferhaus/liste/releases).

### Build from source

```bash
git clone https://github.com/pufferhaus/liste.git
cd liste
go build -o liste .
```

## Quick Start

```bash
# Initialize a roadmap in your project
liste init my-project

# Add items
liste add feature "User authentication"
liste add bug "Login timeout on slow connections"
liste add task "Write integration tests"

# View your roadmap
liste list
liste roadmap

# Work on items
liste set FEAT-001 status active
liste set FEAT-001 priority high
liste set FEAT-001 phase 1
liste done FEAT-001

# Link items
liste link TASK-001 depends-on FEAT-001

# See what's ready to work on
liste next
liste ready

# AI agent context (compact summary for LLM consumption)
liste context
```

![liste demo](demo/liste-demo.gif)

## Interactive TUI

Launch a full-screen interactive TUI with `liste -i`:

```bash
liste -i
```

![liste TUI demo](demo/liste-tui-demo.gif)

### Views

| View | Description |
|------|-------------|
| **list** | All items, scrollable, colour-coded by type and status |
| **roadmap** | Items grouped by phase |
| **blocked** | Items with an active blocker reason |
| **next** | Priority-sorted queue of ready items |
| **search** | Real-time full-text filter |

### Keybindings

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Switch views |
| `↑` / `↓` or `k` / `j` | Navigate list |
| `Enter` | Open item detail overlay |
| `q` / `Escape` | Close detail overlay / quit |
| `d` | Mark selected item done |
| `b` | Block selected item (prompts for reason) |
| `Ctrl+C` | Quit |

The TUI requires a real TTY — `--json` and `--quiet` flags are incompatible with `-i`.

## Concepts

### Item Types

| Type | Prefix | Description |
|------|--------|-------------|
| feature | FEAT- | New capabilities |
| bug | BUG- | Defects to fix |
| idea | IDEA- | Unplanned concepts |
| task | TASK- | Concrete work units |
| epic | EPIC- | Large groupings |

### Statuses

Default lifecycle: `idea` → `planned` → `active` → `done` → `cancelled`

Statuses are configurable per-project in `.liste/config.yaml`.

### Phases

Freeform integers assigned to items. No predefined list — phases are discovered from items. Use them to group work into iterations, milestones, or releases.

```bash
liste phase 1          # Show all items in phase 1
liste roadmap          # Overview grouped by phase
```

### Links

Typed relationships between items:

| Link Type | Inverse | Description |
|-----------|---------|-------------|
| depends-on | blocks | Item cannot proceed until target is done |
| blocks | depends-on | Item is blocking another |
| parent-of | child-of | Hierarchical grouping |
| child-of | parent-of | Belongs to parent |
| relates-to | relates-to | General association |

Links are stored on the declaring item only. Inverse links are resolved at query time — no redundant storage.

### Multi-Project

`liste` discovers nested `.liste/` directories automatically. Use `--project` to scope commands:

```bash
liste list --project transaction-service
liste roadmap   # aggregates all projects
```

## Commands

### Core

| Command | Description |
|---------|-------------|
| `init <name>` | Initialize a new roadmap |
| `add <type> <title>` | Create an item |
| `list` | List items (filterable) |
| `show <id>` | Show item details + links |
| `set <id> <field> <value>` | Update a field |
| `done <id>` | Mark item as done |
| `delete <id>` | Remove an item |

### Status & Planning

| Command | Description |
|---------|-------------|
| `move <id> <status>` | Transition item status |
| `block <id> [reason]` | Mark item blocked |
| `promote <id> <type>` | Re-type an item (e.g. idea → feature) |
| `next` | Show next items to work on |
| `ready` | Show items with resolved dependencies |
| `blocked` | Show all blocked items |
| `stale` | Show items not updated recently |

### Links & Structure

| Command | Description |
|---------|-------------|
| `link <id> <type> <target>` | Create a link |
| `unlink <id> <type> <target>` | Remove a link |
| `graph <id>` | Show item's link graph |
| `tree [id]` | Show hierarchical item tree |

### Views & Reporting

| Command | Description |
|---------|-------------|
| `roadmap` | Phase-grouped overview |
| `phase <n>` | Items in a specific phase |
| `status` | Summary counts by status |
| `progress` | Completion progress |
| `projects` | List discovered projects |
| `context` | Compact AI agent summary |
| `diff --since <date>` | Show changes since a given date |

### Editing

| Command | Description |
|---------|-------------|
| `edit <id>` | Open item in $EDITOR |
| `append <id> <text>` | Append to item body |
| `search <query>` | Full-text search |
| `batch` | Read commands from stdin |

### Skills & TUI

| Command | Description |
|---------|-------------|
| `skills list` | List all bundled Claude Code skills |
| `skills install` | Install skills to `~/.claude` |
| `liste -i` | Launch interactive TUI (requires a TTY) |

## Output Modes

All commands support:

- `--json` — Machine-readable JSON output
- `--quiet` — Minimal output (IDs only)
- Default — Human-readable table format with lipgloss styling
- `-i` / `--interactive` — Full-screen TUI (requires TTY)

`liste add` launched without arguments opens an interactive form when connected to a TTY.

## File Format

Each item is a markdown file with YAML frontmatter:

```markdown
---
id: FEAT-001
type: feature
title: User authentication
status: active
priority: high
phase: 1
created: "2026-01-15"
updated: "2026-05-01"
tags:
    - backend
    - security
links:
    - type: depends-on
      target: TASK-001
blocked:
    reason: Waiting on OAuth provider approval
---

## Description

Implement user authentication with OAuth2 and local credentials.

## Acceptance Criteria

- Users can sign up with email/password
- OAuth2 with Google and GitHub
- Session management with refresh tokens
```

## Configuration

`.liste/config.yaml`:

```yaml
project: my-project
statuses:
    - idea
    - planned
    - active
    - done
    - cancelled
blocked: true
types:
    - feature
    - bug
    - idea
    - task
    - epic
priorities:
    - critical
    - high
    - medium
    - low
defaults:
    status: idea
    priority: medium
```

## Design for AI Agents

`liste` is purpose-built for AI coding agents:

- **Deterministic IDs** — Sequential, predictable (FEAT-001, BUG-002)
- **`--json` everywhere** — Structured output for programmatic consumption
- **`--quiet` mode** — Token-efficient ID-only output
- **`liste context`** — Single command for complete project state (capped at 5 ready items)
- **`liste batch`** — Pipe multiple commands via stdin for atomic multi-mutation
- **`liste next`** — Priority-sorted queue of what to work on next
- **No interactive prompts** — Every operation is fully non-interactive

### Claude Code Skills

liste ships 19 Claude Code skills that teach agents how to use every command. Install them once:

```bash
liste skills install
```

Then add to your `.claude/CLAUDE.md` or `~/.claude/CLAUDE.md`:

```
At the start of every session, invoke the liste:session-start skill.
```

Available skills: `/liste-add-feature`, `/liste-add-bug`, `/liste-add-task`, `/liste-add-idea`, `/liste-add-epic`, `/liste-start`, `/liste-done`, `/liste-block`, `/liste-promote`, `/liste-link`, `/liste-find`, `/liste-append`, `/liste-set`, `/liste-status`, `/liste-next`, `/liste-progress`, `/liste-diff`, `/liste-batch`, `/liste-session-start`

## License

MIT — see [LICENSE](LICENSE).

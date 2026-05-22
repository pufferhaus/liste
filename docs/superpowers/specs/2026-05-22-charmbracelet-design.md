# Charmbracelet Integration Design

**Date:** 2026-05-22
**Scope:** Add lipgloss, glamour, huh, and bubbletea/bubbles to the liste CLI

## Summary

Add charmbracelet tooling to liste in four independent, incrementally shippable layers:

1. **lipgloss** — styled table output (symbols + color text, no background pills)
2. **glamour** — markdown rendering in `liste show`
3. **huh** — TTY-gated interactive forms for `liste add` and `liste init`
4. **bubbletea + bubbles** — full TUI via `liste -i`

Agent-facing paths (JSON, quiet) are untouched throughout. Each layer can be shipped and rolled back independently.

## Approach

Incremental (Approach A). No new abstractions beyond `internal/tui/`. All other changes are in-place edits to existing files. The existing `Formatter` struct in `internal/output/output.go` is the natural boundary for lipgloss and glamour. The `-i` global flag on the root cobra command gates the TUI.

## Architecture

```
internal/
  output/output.go       ← lipgloss styles + glamour render (table mode only)
  tui/
    app.go               ← bubbletea root model, tab state, keybindings
    views/
      list.go            ← bubbles list component (all items)
      roadmap.go         ← phase-grouped roadmap view
      blocked.go         ← blocked items view
      next.go            ← priority-sorted next queue
      search.go          ← text input + filtered results
    detail.go            ← viewport overlay for item detail
cmd/
  add.go                 ← huh form when no args + TTY detected
  init.go                ← huh form when name missing + TTY detected
  root.go                ← -i persistent flag wired to tui.Run()
internal/model/
  project.go             ← TUIConfig field added to Config struct
```

## Section 1: lipgloss styling

**File:** `internal/output/output.go`

A `styles` block defined once at package level. Applied in `FormatTable` path only. JSON and quiet paths are completely untouched.

**Status symbols + colors (scheme B — symbols + color text):**
```
● active     → green  (#00b894)
⊘ blocked    → orange (#e17055)
○ planned    → blue   (#74b9ff)
✓ done       → dimmed gray
✗ cancelled  → dimmed gray
```

**Type symbols:**
```
■ feature  → purple  (#a29bfe)
■ bug      → red     (#ff7675)
■ task     → cyan    (#81ecec)
■ idea     → yellow  (#ffeaa7)
■ epic     → bold white
```

**Priority text:**
```
critical → bold red   (#d63031)
high     → yellow     (#fdcb6e)
medium   → default
low      → dimmed
```

**roadmap.go** — `renderRoadmapTable` and `renderPhaseDetail` delegated to new `Formatter` methods so they pick up lipgloss styles. Phase headers rendered bold with a subtle separator line. `renderRoadmapJSON` stays in `roadmap.go` — only table paths move to the Formatter.

**NO_COLOR** — respected automatically via lipgloss's built-in env var detection. Degrades to plain text with no code changes needed.

## Section 2: glamour markdown rendering

**File:** `internal/output/output.go`, `ItemDetail` method

When `FormatTable` and `item.Body != ""`:
- Pass body through `glamour.Render(body, "auto")` — auto detects dark/light terminal background, falls back to dark
- On any glamour error, fall back silently to raw body text
- No changes to JSON or quiet paths
- No changes to any other command

**Dependency:** `github.com/charmbracelet/glamour`

## Section 3: huh interactive forms

**TTY detection:** `isatty.IsTerminal(os.Stdin.Fd())` — both commands check this before showing any form.

### `liste add`

Current: requires `<type> <title>` positional args.

When both args missing + TTY detected, shows huh form:
```
Type     → huh.Select  (feature / bug / task / idea / epic)
Title    → huh.Input   (required, non-empty)
Priority → huh.Select  (critical / high / medium / low), default: medium
Phase    → huh.Input   (optional, validates as positive integer or empty)
Tags     → huh.Input   (optional, comma-separated, split into []string)
```

When args are present → existing non-interactive path, unchanged.
Partial args (type only, no title) → existing error behavior, unchanged.

### `liste init`

Current: requires `<name>` positional arg.

When name missing + TTY detected, shows huh form:
```
Project name → huh.Input (required, non-empty)
```

When arg is present → existing path, unchanged.

**Dependencies:** `github.com/charmbracelet/huh`, `github.com/mattn/go-isatty`

## Section 4: bubbletea TUI (`liste -i`)

### Entry point

`-i` is a persistent flag on the root cobra command (`cmd/root.go`). When set, `liste -i` (any subcommand or none) calls `tui.Run(store, config)` instead of the normal command path. `-i` and `--json`/`--quiet` are mutually exclusive — error out with a clear message if both are set.

### Config

New `TUIConfig` field in `internal/model/project.go`:

```go
type TUIConfig struct {
    DefaultView string   `yaml:"default_view"` // list | roadmap | blocked | next | search
    Views       []string `yaml:"views"`         // ordered list of enabled views
}
```

Added to `Config` struct as `TUI TUIConfig yaml:"tui"`.

**Defaults when absent:**
- `DefaultView`: `"list"`
- `Views`: `["list", "roadmap", "blocked", "next", "search"]`

**`.liste/config.yaml` example:**
```yaml
tui:
  default_view: roadmap
  views: [roadmap, list, next, blocked, search]
```

### Root model (`internal/tui/app.go`)

Bubbletea `Model` holds:
- `tabs []string` — view names in configured order
- `activeTab int` — current tab index (starts at DefaultView index)
- `views map[string]tea.Model` — lazily initialized on first visit
- `overlay *detailModel` — non-nil when detail overlay is open
- `store *store.Store` — passed in, shared across all views
- `width, height int` — terminal dimensions from `tea.WindowSizeMsg`

### Views

Each view in `internal/tui/views/`:

| View | Component | Data source |
|---|---|---|
| `list.go` | bubbles `list` | `store.ListItems()` |
| `roadmap.go` | custom bubbletea model | same logic as `cmd/roadmap.go` |
| `blocked.go` | bubbles `list` | `store.ListItems()` filtered to blocked |
| `next.go` | bubbles `list` | same logic as `cmd/next.go` |
| `search.go` | bubbles `textinput` + filtered list | `store.ListItems()`, client-side filter |

No duplicate store/business logic — views call the same store methods as the CLI commands.

### Detail overlay (`internal/tui/detail.go`)

- bubbles `viewport` containing full item detail
- Body rendered via glamour (same call as `ItemDetail`)
- Opens on `enter`, closes on `esc`
- Rendered on top of the current view (overlay pattern)

### Keybindings

```
tab / shift+tab    cycle views
↑ / k              navigate up
↓ / j              navigate down
enter              open detail overlay
esc                close overlay / clear search filter
d                  mark current item done (writes through store)
b                  mark current item blocked (huh form embedded as bubbletea model for reason input)
/                  focus search input (search view) / filter (list views)
q / ctrl+c         quit
```

Mutations (`d`, `b`) use the existing store write methods — no duplicate logic.

### Dependencies

```
github.com/charmbracelet/bubbletea
github.com/charmbracelet/bubbles
github.com/charmbracelet/lipgloss  (already added in Section 1)
```

## New dependencies summary

```
github.com/charmbracelet/lipgloss
github.com/charmbracelet/glamour
github.com/charmbracelet/huh
github.com/charmbracelet/bubbletea
github.com/charmbracelet/bubbles
github.com/mattn/go-isatty
```

## What does NOT change

- `--json` output: zero changes
- `--quiet` output: zero changes
- `liste batch`, `liste context`: zero changes (AI agent paths)
- All existing positional arg behavior when args are present
- File format (`.liste/` markdown files): zero changes
- Existing config fields: additive only (`tui:` is a new top-level key)

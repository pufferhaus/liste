# Manual Testing Checklist

End-to-end smoke test for the `liste` CLI + TUI. Run this before releases
or whenever you want to verify nothing has regressed.

CLI tests are scripted and can be re-run with `bash dev/manual-test.sh`
(see bottom of this file). TUI tests require a real TTY and a human
driver — checklist included.

---

## 0. Pre-flight

```bash
# Build fresh dev binary with proper version string
VER="dev+$(git rev-parse --short HEAD)"
go build -ldflags "-s -w -X main.version=$VER -X main.commit=$(git rev-parse --short HEAD) -X main.date=$(date -u +%Y-%m-%d)" -o ~/.local/bin/liste .

# Sanity checks
liste -v                          # bare version
liste version                     # full version line
go test ./...                     # 55+ passing
~/go/bin/golangci-lint run        # exit 0
```

---

## 1. Sandbox setup

```bash
rm -rf /tmp/liste-sandbox
mkdir -p /tmp/liste-sandbox/services/api
cd /tmp/liste-sandbox && liste init demoroot
cd /tmp/liste-sandbox/services/api && liste init api-service
cd /tmp/liste-sandbox
```

Expected: two `.liste/` directories with `config.yaml` and `.state.yaml`.

---

## 2. Core mutations

| # | Command | Expected |
|---|---------|----------|
| 2.1 | `liste add feature "OAuth login"` | `Created FEAT-001` |
| 2.2 | `liste add feature "User profile page"` | `Created FEAT-002` |
| 2.3 | `liste add bug "Login timeout on Safari"` | `Created BUG-001` |
| 2.4 | `liste add bug "Password reset email malformed"` | `Created BUG-002` |
| 2.5 | `liste add task "Write integration tests"` | `Created TASK-001` |
| 2.6 | `liste add task "Document API endpoints"` | `Created TASK-002` |
| 2.7 | `liste add idea "Dark mode"` | `Created IDEA-001` |
| 2.8 | `liste add epic "Auth platform"` | `Created EPIC-001` |
| 2.9 | `liste list` | 8 items, sorted |
| 2.10 | `liste show FEAT-001` | full detail view |
| 2.11 | `liste set FEAT-001 priority high` | `Updated FEAT-001: priority = high` |
| 2.12 | `liste set FEAT-001 phase 1` | `Updated FEAT-001: phase = 1` |
| 2.13 | `liste set FEAT-001 status active` | `Updated FEAT-001: status = active` |
| 2.14 | `liste set BUG-001 priority critical` | `Updated BUG-001: priority = critical` |
| 2.15 | `liste set EPIC-001 phase 1` | `Updated EPIC-001: phase = 1` |
| 2.16 | `liste move FEAT-002 planned` | `Moved FEAT-002 to planned` |
| 2.17 | `liste block BUG-002 "Waiting on SMTP creds"` | `Blocked BUG-002` |
| 2.18 | `liste promote IDEA-001 feature` | `Promoted IDEA-001 -> FEAT-003` |
| 2.19 | `liste done TASK-002` | `Marked TASK-002 as done` |

---

## 3. Links and structure

| # | Command | Expected |
|---|---------|----------|
| 3.1 | `liste link TASK-001 depends-on FEAT-001` | `Linked TASK-001 -[depends-on]-> FEAT-001` |
| 3.2 | `liste link FEAT-001 child-of EPIC-001` | `Linked FEAT-001 -[child-of]-> EPIC-001` |
| 3.3 | `liste link FEAT-002 child-of EPIC-001` | `Linked FEAT-002 -[child-of]-> EPIC-001` |
| 3.4 | `liste link FEAT-001 relates-to BUG-001` | `Linked FEAT-001 -[relates-to]-> BUG-001` |
| 3.5 | `liste graph FEAT-001` | shows 3 outgoing links + 1 inverse (`blocks TASK-001`) |
| 3.6 | `liste tree EPIC-001` | tree of EPIC-001 with FEAT-001 and FEAT-002 |
| 3.7 | `liste unlink FEAT-001 BUG-001` | `Removed link(s) from FEAT-001 to BUG-001` (see KNOWN ISSUES — signature is 2-arg, README claims 3-arg) |

---

## 4. Views and reporting

| # | Command | Expected |
|---|---------|----------|
| 4.1 | `liste list --type bug` | only BUG-* |
| 4.2 | `liste list --status active` | only `active` items |
| 4.3 | `liste list --priority high` | only `high` items |
| 4.4 | `liste ready` | items with no unmet dependencies |
| 4.5 | `liste next` | single top-priority item |
| 4.6 | `liste next --count 3` | top 3 |
| 4.7 | `liste blocked` | `BUG-002` with reason |
| 4.8 | `liste roadmap` | phase-grouped view |
| 4.9 | `liste phase 1` | items in phase 1 |
| 4.10 | `liste status` | status-grouped counts + items |
| 4.11 | `liste progress` | overall + per-phase % bars |
| 4.12 | `liste projects` | root + sub-project list |
| 4.13 | `liste context` | compact AI summary |
| 4.14 | `liste stale` | `No stale items` (defaults to 14-day threshold) |
| 4.15 | `liste diff` | items created since yesterday |
| 4.16 | `liste diff --since 2026-01-01` | items created since date |

---

## 5. Editing

| # | Command | Expected |
|---|---------|----------|
| 5.1 | `EDITOR=true liste edit TASK-001` | silently opens & closes |
| 5.2 | `liste append FEAT-001 "Decided to use OAuth2.0 with PKCE flow"` | note appended; `show` reflects it |
| 5.3 | `liste search "OAuth"` | matches `FEAT-001` |
| 5.4 | `liste search "OAuth" --quiet` | bare ID(s) only |
| 5.5 | `liste search "OAuth" --json` | structured JSON array |
| 5.6 | `liste batch <<EOF` ... `EOF` | multi-mutation: `Batch complete: N executed, 0 errors` |

```bash
liste batch <<EOF
add bug "Test bug from batch"
add task "Test task from batch"
set BUG-001 phase 2
EOF
```

---

## 6. Output flags

| # | Command | Expected |
|---|---------|----------|
| 6.1 | `liste add bug "Quick capture" --quiet` | bare new ID, nothing else |
| 6.2 | `liste list --quiet` | one ID per line |
| 6.3 | `liste list --json` | JSON array of items |
| 6.4 | `liste status --json` | `{by_status, items}` JSON object |
| 6.5 | `liste next --json` | item JSON |
| 6.6 | `liste progress --json` | `{overall_percent, total_done, total_items, phases}` |
| 6.7 | `liste diff --since 2026-01-01 --json` | `{created, updated, completed}` arrays |

---

## 7. Multi-project

| # | Command | Expected |
|---|---------|----------|
| 7.1 | `liste add task "Rate limit" --project services/api` | `Created TASK-001` (note: use PATH not project name — see KNOWN ISSUES) |
| 7.2 | `liste list --project services/api` | only sub-project items |
| 7.3 | `liste roadmap` | aggregates both projects, items grouped under project name |
| 7.4 | `liste context` | shows multi-project totals |

---

## 8. Delete

| # | Command | Expected |
|---|---------|----------|
| 8.1 | `liste delete TASK-X` | confirmation prompt: `Use --force to confirm` |
| 8.2 | `liste delete TASK-X --force` | `Deleted TASK-X` |

---

## 9. Skills

| # | Command | Expected |
|---|---------|----------|
| 9.1 | `liste skills list` | 19 skills enumerated |
| 9.2 | `liste skills install` | installs to `~/.claude/plugins/cache/liste/liste/<version>/` and updates `installed_plugins.json` |

After install, verify:
```bash
grep -A6 '"liste@liste"' ~/.claude/plugins/installed_plugins.json
```
Version field should match the running binary.

---

## 10. TUI

The TUI is covered by an automated `teatest`-based suite in `internal/tui/`:

```bash
go test ./internal/tui/... -race -timeout=120s
```

Coverage (one file per area, all keyboard + mouse paths):

| File | Tests |
|------|-------|
| `tab_test.go` | tab bar render, forward/backward cycle, mouse click on tab |
| `detail_test.go` | enter opens detail, action-bar key + mouse, `d`/`b`/`e`/esc |
| `modal_test.go` | delete-confirm modal: keyboard `y`/`n`/esc + mouse confirm/cancel |
| `edit_test.go` | open edit, tab-cycle fields, save (ctrl+s), discard modal y/n |
| `search_test.go` | filter-as-you-type, esc clears, enter + mouse open detail |
| `lifecycle_test.go` | `q`/ctrl+c quit, window resize propagation |

Add a new test alongside the matching file when adding a TUI feature.

### Manual-only checks (need a real TTY)

These don't get automated coverage:

- [ ] Banner renders with logo + version on `liste -i` startup
- [ ] Trying `--json` with `-i`: `Error: --interactive cannot be used with --json or --quiet`
- [ ] Trying `--quiet` with `-i`: same error
- [ ] AltScreen + mouse-cell-motion enable cleanly on startup, restore on quit
- [ ] Glamour markdown rendering in detail body visually sane on real terminals

---

## KNOWN ISSUES (found while building this checklist, 2026-05-23)

These should be filed in `liste` itself (`liste add bug ...`):

1. **`liste tree` without an ID errors**, despite README + skills suggesting `liste tree` shows the whole hierarchy. Either fix the command to accept no-arg form or update docs.
2. **`liste unlink` signature mismatch**: README + skills say `liste unlink <id> <type> <target>` (3 args). Actual binary takes `<id> <target>` (2 args) and removes all links matching the (id, target) pair regardless of type.
3. **`liste diff` never populates `completed`**: items marked `done` are not surfaced under COMPLETED in `liste diff` output. Likely because `done` doesn't write a `completed:` timestamp; diff filter requires it.
4. **`--project` requires the path, not the project name**: `liste list --project api-service` errors with `project not found` even though `liste projects` displays `api-service` in its config. Have to use `--project services/api`. Either resolver should accept either form, or `liste projects` should display the path users need to pass.
5. **Edit-from-detail input routing was broken pre-2026-05-23**: when the edit overlay was opened from the detail view, keystrokes were swallowed by the still-active detail's viewport instead of reaching the edit form's textinputs. Fixed in `app.go` by prioritizing `editOverlay` over `overlay` in the input router (originally exposed by the new teatest suite). Save (`ctrl+s`) also no longer worked from that flow because `ItemSavedMsg` reached the edit overlay instead of `AppModel`; promoted to an early intercept.

---

## Re-running

After fixes ship, re-run sections 1–9 from the top. Bash one-liner version
of the CLI portion is kept in `dev/manual-test.sh` (TODO: extract).

When new features land, add a row to the appropriate section and a test
alongside the matching file in `internal/tui/`.

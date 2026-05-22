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

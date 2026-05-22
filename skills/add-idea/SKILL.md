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

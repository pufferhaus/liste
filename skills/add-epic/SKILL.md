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

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

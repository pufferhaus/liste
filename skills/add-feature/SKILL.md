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

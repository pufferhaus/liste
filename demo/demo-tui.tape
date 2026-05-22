# liste TUI demo
Output demo/liste-tui-demo.gif

Require liste

Set Shell zsh
Set FontSize 14
Set Width 900
Set Height 560
Set Theme "Catppuccin Mocha"
Set TypingSpeed 3ms
Set Padding 20

# Rapid setup
Type "DEMO=$(mktemp -d) && cd $DEMO && liste init my-project"
Enter
Sleep 600ms
Type "liste add feature 'OAuth2 login support'"
Enter
Sleep 100ms
Type "liste add task 'Set up Postgres schema'"
Enter
Sleep 100ms
Type "liste add bug 'Session expires too early'"
Enter
Sleep 100ms
Type "liste add epic 'v1 launch'"
Enter
Sleep 100ms
Type "liste add feature 'Dark mode support'"
Enter
Sleep 100ms
Type "liste add task 'Write API docs'"
Enter
Sleep 100ms
Type "liste add idea 'Mobile app'"
Enter
Sleep 100ms
Type "liste set FEAT-001 status active && liste set FEAT-001 priority high && liste set FEAT-001 phase 1"
Enter
Sleep 100ms
Type "liste link EPIC-001 parent-of FEAT-001 && liste block BUG-001 'waiting for repro'"
Enter
Sleep 400ms

# Slow down for TUI
Set TypingSpeed 60ms

Type "liste -i"
Enter
Sleep 1800ms

# LIST view — navigate to second item
Down
Sleep 600ms
Down
Sleep 600ms

# Open detail overlay
Enter
Sleep 2200ms

# Close detail with q (Detail handles q → CloseDetailMsg)
Type "q"
Sleep 1000ms

# Navigate to see different items
Down
Sleep 500ms
Down
Sleep 500ms

# TAB → roadmap view
Tab
Sleep 1800ms

# Scroll roadmap a bit
Down
Sleep 400ms
Down
Sleep 400ms

# TAB → blocked view
Tab
Sleep 1800ms

# TAB → next view  
Tab
Sleep 1800ms

# TAB → search view
Tab
Sleep 1800ms

# Type search query — filters in real time
Type "auth"
Sleep 1500ms

# Clear with esc
Escape
Sleep 800ms

# Search again
Type "dark"
Sleep 1500ms

# Clear
Escape
Sleep 600ms

# Quit from search (q is global, handled before text input sees it)
Ctrl+C
Sleep 800ms

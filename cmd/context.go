package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/pufferhaus/liste/internal/model"
	"github.com/spf13/cobra"
)

var (
	contextPhase int
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Emit a compact summary for AI agent context injection",
	Long: `Produces a token-efficient summary of the current roadmap state designed
to be injected into an AI agent's system prompt or context window.

Includes: current phase, active items, blockers, ready queue, and key stats.
Output is ~30-50 lines maximum.`,
	Args: cobra.NoArgs,
	RunE: runContext,
}

func init() {
	contextCmd.Flags().IntVar(&contextPhase, "phase", 0, "Focus on a specific phase")
	rootCmd.AddCommand(contextCmd)
}

// contextData holds the computed state for rendering.
type contextData struct {
	ProjectName   string
	TotalItems    int
	ActivePhase   int
	PhasesDone    int
	PhasesTotal   int
	ActiveItems   []contextItem
	BlockedItems  []contextItem
	ReadyQueue    []contextItem
	PhasesSummary []contextPhaseSummary
}

type contextItem struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	Priority string `json:"priority"`
	Phase    *int   `json:"phase,omitempty"`
	Project  string `json:"project"`
	Reason   string `json:"reason,omitempty"`
}

type contextPhaseSummary struct {
	Phase  int    `json:"phase"`
	Status string `json:"status"`
	Done   int    `json:"done"`
	Total  int    `json:"total"`
}

func runContext(cmd *cobra.Command, args []string) error {
	allEntries, allItemsByID, rootProject, err := collectAllItems()
	if err != nil {
		return err
	}

	data := computeContext(allEntries, allItemsByID, rootProject)

	if contextPhase > 0 {
		data = filterContextToPhase(data, allEntries, allItemsByID, contextPhase)
	}

	if flagJSON {
		renderContextJSON(data)
		return nil
	}

	renderContextText(data)
	return nil
}

func computeContext(entries []itemWithProject, allItems map[string]*model.Item, rootProject string) contextData {
	data := contextData{
		ProjectName: rootProject,
		TotalItems:  len(entries),
	}

	// Compute phase stats
	phaseItems := make(map[int][]itemWithProject)
	for _, entry := range entries {
		if entry.item.Phase != nil {
			phaseItems[*entry.item.Phase] = append(phaseItems[*entry.item.Phase], entry)
		}
	}

	var phaseNums []int
	for p := range phaseItems {
		phaseNums = append(phaseNums, p)
	}
	sort.Ints(phaseNums)

	data.PhasesTotal = len(phaseNums)
	activePhaseFound := false

	for _, p := range phaseNums {
		items := phaseItems[p]
		done := 0
		total := len(items)
		hasActive := false
		for _, entry := range items {
			if entry.item.Status == "done" || entry.item.Status == "cancelled" {
				done++
			}
			if entry.item.Status == "active" {
				hasActive = true
			}
		}

		status := "upcoming"
		if done == total {
			status = "complete"
			data.PhasesDone++
		} else if hasActive && !activePhaseFound {
			status = "active"
			data.ActivePhase = p
			activePhaseFound = true
		} else if !activePhaseFound && data.ActivePhase == 0 {
			data.ActivePhase = p
		}

		data.PhasesSummary = append(data.PhasesSummary, contextPhaseSummary{
			Phase:  p,
			Status: status,
			Done:   done,
			Total:  total,
		})
	}

	// Collect active, blocked, and ready items
	for _, entry := range entries {
		item := entry.item

		if item.Status == "active" {
			data.ActiveItems = append(data.ActiveItems, contextItem{
				ID:       item.ID,
				Type:     string(item.Type),
				Title:    item.Title,
				Priority: item.Priority,
				Phase:    item.Phase,
				Project:  entry.project,
			})
		}

		if item.Blocked != nil {
			reason := item.Blocked.Reason
			if reason == "" {
				reason = "no reason specified"
			}
			data.BlockedItems = append(data.BlockedItems, contextItem{
				ID:       item.ID,
				Type:     string(item.Type),
				Title:    item.Title,
				Priority: item.Priority,
				Phase:    item.Phase,
				Project:  entry.project,
				Reason:   reason,
			})
		}

		// Ready queue: not done, not active, not blocked, deps resolved
		if item.Status != "done" && item.Status != "cancelled" && item.Status != "active" && item.Blocked == nil {
			if depsResolved(item, allItems) {
				data.ReadyQueue = append(data.ReadyQueue, contextItem{
					ID:       item.ID,
					Type:     string(item.Type),
					Title:    item.Title,
					Priority: item.Priority,
					Phase:    item.Phase,
					Project:  entry.project,
				})
			}
		}
	}

	// Sort ready queue by phase then priority
	sort.SliceStable(data.ReadyQueue, func(i, j int) bool {
		a := data.ReadyQueue[i]
		b := data.ReadyQueue[j]
		aPhase := 9999
		bPhase := 9999
		if a.Phase != nil {
			aPhase = *a.Phase
		}
		if b.Phase != nil {
			bPhase = *b.Phase
		}
		if aPhase != bPhase {
			return aPhase < bPhase
		}
		return priorityWeight(a.Priority) < priorityWeight(b.Priority)
	})

	// Cap ready queue at 5 for context brevity
	if len(data.ReadyQueue) > 5 {
		data.ReadyQueue = data.ReadyQueue[:5]
	}

	return data
}

func filterContextToPhase(data contextData, entries []itemWithProject, allItems map[string]*model.Item, phase int) contextData {
	var filtered []itemWithProject
	for _, entry := range entries {
		if entry.item.Phase != nil && *entry.item.Phase == phase {
			filtered = append(filtered, entry)
		}
	}

	newData := computeContext(filtered, allItems, data.ProjectName)
	newData.PhasesSummary = data.PhasesSummary
	newData.PhasesTotal = data.PhasesTotal
	newData.PhasesDone = data.PhasesDone
	newData.ActivePhase = data.ActivePhase
	return newData
}

func renderContextText(data contextData) {
	fmt.Fprintf(os.Stdout, "# Roadmap Context: %s\n", data.ProjectName)
	fmt.Fprintf(os.Stdout, "Items: %d | Phases: %d/%d complete | Active phase: %d\n\n",
		data.TotalItems, data.PhasesDone, data.PhasesTotal, data.ActivePhase)

	if len(data.PhasesSummary) > 0 {
		fmt.Fprintln(os.Stdout, "## Phases")
		for _, ps := range data.PhasesSummary {
			marker := " "
			switch ps.Status {
			case "complete":
				marker = "x"
			case "active":
				marker = ">"
			}
			fmt.Fprintf(os.Stdout, "  [%s] Phase %d: %d/%d (%s)\n", marker, ps.Phase, ps.Done, ps.Total, ps.Status)
		}
		fmt.Fprintln(os.Stdout)
	}

	if len(data.ActiveItems) > 0 {
		fmt.Fprintln(os.Stdout, "## In Progress")
		for _, item := range data.ActiveItems {
			phaseStr := ""
			if item.Phase != nil {
				phaseStr = fmt.Sprintf(" p%d", *item.Phase)
			}
			fmt.Fprintf(os.Stdout, "  %s [%s]%s %s (%s)\n", item.ID, item.Priority, phaseStr, item.Title, item.Project)
		}
		fmt.Fprintln(os.Stdout)
	}

	if len(data.BlockedItems) > 0 {
		fmt.Fprintln(os.Stdout, "## Blocked")
		for _, item := range data.BlockedItems {
			fmt.Fprintf(os.Stdout, "  %s %s — %s\n", item.ID, item.Title, item.Reason)
		}
		fmt.Fprintln(os.Stdout)
	}

	if len(data.ReadyQueue) > 0 {
		fmt.Fprintln(os.Stdout, "## Ready (next up)")
		for _, item := range data.ReadyQueue {
			phaseStr := ""
			if item.Phase != nil {
				phaseStr = fmt.Sprintf(" p%d", *item.Phase)
			}
			fmt.Fprintf(os.Stdout, "  %s [%s]%s %s (%s)\n", item.ID, item.Priority, phaseStr, item.Title, item.Project)
		}
		fmt.Fprintln(os.Stdout)
	}
}

func renderContextJSON(data contextData) {
	type jsonOutput struct {
		Project     string                `json:"project"`
		TotalItems  int                   `json:"total_items"`
		ActivePhase int                   `json:"active_phase"`
		PhasesDone  int                   `json:"phases_done"`
		PhasesTotal int                   `json:"phases_total"`
		Phases      []contextPhaseSummary `json:"phases"`
		Active      []contextItem         `json:"active"`
		Blocked     []contextItem         `json:"blocked"`
		Ready       []contextItem         `json:"ready"`
	}

	out := jsonOutput{
		Project:     data.ProjectName,
		TotalItems:  data.TotalItems,
		ActivePhase: data.ActivePhase,
		PhasesDone:  data.PhasesDone,
		PhasesTotal: data.PhasesTotal,
		Phases:      data.PhasesSummary,
		Active:      data.ActiveItems,
		Blocked:     data.BlockedItems,
		Ready:       data.ReadyQueue,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

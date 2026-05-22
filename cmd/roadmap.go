package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"

	"github.com/pblca/liste/internal/model"
	"github.com/pblca/liste/internal/output"
	"github.com/pblca/liste/internal/store"
	"github.com/spf13/cobra"
)

var (
	roadmapPhase int
)

var roadmapCmd = &cobra.Command{
	Use:   "roadmap",
	Short: "Show the unified roadmap view across all projects",
	Long:  "Display items organized by phase across all discovered projects. Complete phases are collapsed.",
	Args:  cobra.NoArgs,
	RunE:  runRoadmap,
}

func init() {
	roadmapCmd.Flags().IntVar(&roadmapPhase, "phase", 0, "Show only a specific phase number")
	rootCmd.AddCommand(roadmapCmd)
}

// phaseGroup holds all items for a given phase, grouped by project.
type phaseGroup struct {
	Phase    int
	Projects []projectItems
}

// projectItems holds items for a single project within a phase.
type projectItems struct {
	Name  string
	Items []*model.Item
}

// phaseStatus represents the auto-detected status of a phase.
type phaseStatus string

const (
	phaseComplete phaseStatus = "complete"
	phaseActive   phaseStatus = "active"
	phaseUpcoming phaseStatus = "upcoming"
)

func runRoadmap(cmd *cobra.Command, args []string) error {
	result, err := getDiscovery()
	if err != nil {
		return err
	}

	// Collect items from all projects
	type projectData struct {
		name  string
		items []*model.Item
	}

	var allProjects []projectData

	// Root project
	rootStore := store.New(result.Root)
	rootCfg, err := rootStore.ReadConfig()
	if err != nil {
		return err
	}
	rootItems, err := rootStore.ListItems()
	if err != nil {
		return err
	}
	allProjects = append(allProjects, projectData{name: rootCfg.Project, items: rootItems})

	// Sub-projects
	for _, sub := range result.SubProjects {
		s := store.New(sub.Path)
		items, err := s.ListItems()
		if err != nil {
			continue
		}
		allProjects = append(allProjects, projectData{name: sub.Name, items: items})
	}

	// Group items by phase
	phaseMap := make(map[int]map[string][]*model.Item) // phase -> project name -> items
	var unphased []projectItems

	for _, proj := range allProjects {
		var unphasedItems []*model.Item
		for _, item := range proj.items {
			if item.Phase == nil {
				unphasedItems = append(unphasedItems, item)
			} else {
				phase := *item.Phase
				if phaseMap[phase] == nil {
					phaseMap[phase] = make(map[string][]*model.Item)
				}
				phaseMap[phase][proj.name] = append(phaseMap[phase][proj.name], item)
			}
		}
		if len(unphasedItems) > 0 {
			unphased = append(unphased, projectItems{Name: proj.name, Items: unphasedItems})
		}
	}

	// Sort phases
	var phaseNums []int
	for p := range phaseMap {
		phaseNums = append(phaseNums, p)
	}
	sort.Ints(phaseNums)

	// Build phase groups
	var phases []phaseGroup
	for _, p := range phaseNums {
		projMap := phaseMap[p]
		var projs []projectItems
		for _, proj := range allProjects {
			if items, ok := projMap[proj.name]; ok {
				projs = append(projs, projectItems{Name: proj.name, Items: items})
			}
		}
		phases = append(phases, phaseGroup{Phase: p, Projects: projs})
	}

	// If filtering to a specific phase, delegate to phase detail view
	if roadmapPhase > 0 {
		return renderPhaseDetail(phases, roadmapPhase)
	}

	// Determine phase statuses
	if flagJSON {
		renderRoadmapJSON(phases, unphased)
		return nil
	}

	// Render roadmap view
	renderRoadmapTable(phases, unphased)
	return nil
}

// detectPhaseStatus determines the auto-detected status of a phase.
func detectPhaseStatus(pg phaseGroup) phaseStatus {
	total := 0
	done := 0
	hasActive := false

	for _, proj := range pg.Projects {
		for _, item := range proj.Items {
			total++
			if item.Status == "done" || item.Status == "cancelled" {
				done++
			}
			if item.Status == "active" {
				hasActive = true
			}
		}
	}

	if total > 0 && done == total {
		return phaseComplete
	}
	if hasActive {
		return phaseActive
	}
	return phaseUpcoming
}

// phaseProgress returns done/total counts for a phase.
func phaseGroupProgress(pg phaseGroup) (int, int) {
	total := 0
	done := 0
	for _, proj := range pg.Projects {
		for _, item := range proj.Items {
			total++
			if item.Status == "done" || item.Status == "cancelled" {
				done++
			}
		}
	}
	return done, total
}

func renderRoadmapTable(phases []phaseGroup, unphased []projectItems) {
	for _, pg := range phases {
		status := detectPhaseStatus(pg)
		done, total := phaseGroupProgress(pg)

		if status == phaseComplete {
			fmt.Fprintln(os.Stdout, output.RenderPhaseHeader(pg.Phase, "complete", done, total))
			fmt.Fprintln(os.Stdout)
			continue
		}

		fmt.Fprintln(os.Stdout, output.RenderPhaseHeader(pg.Phase, string(status), done, total))
		for _, proj := range pg.Projects {
			if hasMultipleProjects(phases, unphased) {
				fmt.Fprintf(os.Stdout, "  %s\n", proj.Name)
			}
			for _, item := range proj.Items {
				indent := "  "
				if hasMultipleProjects(phases, unphased) {
					indent = "    "
				}
				fmt.Fprintf(os.Stdout, "%s%s  %-10s  %s\n",
					indent,
					output.RenderStatus(item.Status, item.Blocked != nil),
					item.ID,
					item.Title,
				)
			}
		}
		fmt.Fprintln(os.Stdout)
	}

	if len(unphased) > 0 {
		totalUnphased := 0
		for _, proj := range unphased {
			totalUnphased += len(proj.Items)
		}
		fmt.Fprintf(os.Stdout, "%s\n", output.RenderPhaseHeader(0, "unphased", 0, totalUnphased))
		for _, proj := range unphased {
			if hasMultipleProjects(phases, unphased) {
				fmt.Fprintf(os.Stdout, "  %s\n", proj.Name)
			}
			for _, item := range proj.Items {
				indent := "  "
				if hasMultipleProjects(phases, unphased) {
					indent = "    "
				}
				fmt.Fprintf(os.Stdout, "%s%s  %-10s  %s\n",
					indent,
					output.RenderStatus(item.Status, item.Blocked != nil),
					item.ID,
					item.Title,
				)
			}
		}
		fmt.Fprintln(os.Stdout)
	}
}

// hasMultipleProjects checks if there are items from more than one project.
func hasMultipleProjects(phases []phaseGroup, unphased []projectItems) bool {
	projects := make(map[string]bool)
	for _, pg := range phases {
		for _, proj := range pg.Projects {
			projects[proj.Name] = true
		}
	}
	for _, proj := range unphased {
		projects[proj.Name] = true
	}
	return len(projects) > 1
}

func renderRoadmapJSON(phases []phaseGroup, unphased []projectItems) {
	type jsonItem struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Title    string `json:"title"`
		Status   string `json:"status"`
		Priority string `json:"priority"`
		Blocked  bool   `json:"blocked,omitempty"`
	}
	type jsonProject struct {
		Name  string     `json:"name"`
		Items []jsonItem `json:"items"`
	}
	type jsonPhase struct {
		Phase    int           `json:"phase"`
		Status   string        `json:"status"`
		Done     int           `json:"done"`
		Total    int           `json:"total"`
		Projects []jsonProject `json:"projects"`
	}

	var jPhases []jsonPhase
	for _, pg := range phases {
		done, total := phaseGroupProgress(pg)
		var jProjects []jsonProject
		for _, proj := range pg.Projects {
			var jItems []jsonItem
			for _, item := range proj.Items {
				jItems = append(jItems, jsonItem{
					ID:       item.ID,
					Type:     string(item.Type),
					Title:    item.Title,
					Status:   item.Status,
					Priority: item.Priority,
					Blocked:  item.Blocked != nil,
				})
			}
			jProjects = append(jProjects, jsonProject{Name: proj.Name, Items: jItems})
		}
		jPhases = append(jPhases, jsonPhase{
			Phase:    pg.Phase,
			Status:   string(detectPhaseStatus(pg)),
			Done:     done,
			Total:    total,
			Projects: jProjects,
		})
	}

	// Unphased
	var jUnphased []jsonProject
	for _, proj := range unphased {
		var jItems []jsonItem
		for _, item := range proj.Items {
			jItems = append(jItems, jsonItem{
				ID:       item.ID,
				Type:     string(item.Type),
				Title:    item.Title,
				Status:   item.Status,
				Priority: item.Priority,
				Blocked:  item.Blocked != nil,
			})
		}
		jUnphased = append(jUnphased, jsonProject{Name: proj.Name, Items: jItems})
	}

	result := map[string]any{
		"phases":   jPhases,
		"unphased": jUnphased,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(result)
}

func renderPhaseDetail(phases []phaseGroup, phaseNum int) error {
	var target *phaseGroup
	for i := range phases {
		if phases[i].Phase == phaseNum {
			target = &phases[i]
			break
		}
	}

	if target == nil {
		return fmt.Errorf("no items found in phase %d", phaseNum)
	}

	done, total := phaseGroupProgress(*target)
	status := detectPhaseStatus(*target)

	if flagJSON {
		// JSON output for phase detail
		type jsonItem struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Title    string `json:"title"`
			Status   string `json:"status"`
			Priority string `json:"priority"`
			Blocked  bool   `json:"blocked,omitempty"`
		}
		type jsonProject struct {
			Name  string     `json:"name"`
			Items []jsonItem `json:"items"`
		}

		var jProjects []jsonProject
		for _, proj := range target.Projects {
			var jItems []jsonItem
			for _, item := range proj.Items {
				jItems = append(jItems, jsonItem{
					ID:       item.ID,
					Type:     string(item.Type),
					Title:    item.Title,
					Status:   item.Status,
					Priority: item.Priority,
					Blocked:  item.Blocked != nil,
				})
			}
			jProjects = append(jProjects, jsonProject{Name: proj.Name, Items: jItems})
		}

		result := map[string]any{
			"phase":    phaseNum,
			"status":   string(status),
			"done":     done,
			"total":    total,
			"projects": jProjects,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(result)
		return nil
	}

	// Table output
	fmt.Fprintln(os.Stdout, output.RenderPhaseHeader(phaseNum, string(status), done, total))
	fmt.Fprintln(os.Stdout)
	for _, proj := range target.Projects {
		fmt.Fprintf(os.Stdout, "  %s\n", proj.Name)
		for _, item := range proj.Items {
			fmt.Fprintf(os.Stdout, "    %s  %-10s  %s  %s\n",
				output.RenderStatus(item.Status, item.Blocked != nil),
				item.ID,
				output.RenderPriority(item.Priority),
				item.Title,
			)
		}
		fmt.Fprintln(os.Stdout)
	}

	return nil
}

// phaseCmd provides a shortcut to view a specific phase.
var phaseCmd = &cobra.Command{
	Use:   "phase <number>",
	Short: "Show detail view of a specific phase",
	Long:  "Display all items in a specific phase across all projects with full detail.",
	Args:  cobra.ExactArgs(1),
	RunE:  runPhase,
}

func init() {
	rootCmd.AddCommand(phaseCmd)
}

func runPhase(cmd *cobra.Command, args []string) error {
	phaseNum, err := strconv.Atoi(args[0])
	if err != nil || phaseNum < 1 {
		return fmt.Errorf("phase must be a positive integer")
	}

	result, err := getDiscovery()
	if err != nil {
		return err
	}

	// Collect items from all projects
	type projectData struct {
		name  string
		items []*model.Item
	}

	var allProjects []projectData

	rootStore := store.New(result.Root)
	rootCfg, err := rootStore.ReadConfig()
	if err != nil {
		return err
	}
	rootItems, err := rootStore.ListItems()
	if err != nil {
		return err
	}
	allProjects = append(allProjects, projectData{name: rootCfg.Project, items: rootItems})

	for _, sub := range result.SubProjects {
		s := store.New(sub.Path)
		items, err := s.ListItems()
		if err != nil {
			continue
		}
		allProjects = append(allProjects, projectData{name: sub.Name, items: items})
	}

	// Build just the target phase
	projMap := make(map[string][]*model.Item)
	for _, proj := range allProjects {
		for _, item := range proj.items {
			if item.Phase != nil && *item.Phase == phaseNum {
				projMap[proj.name] = append(projMap[proj.name], item)
			}
		}
	}

	if len(projMap) == 0 {
		return fmt.Errorf("no items found in phase %d", phaseNum)
	}

	// Build phase group preserving project order
	var projs []projectItems
	for _, proj := range allProjects {
		if items, ok := projMap[proj.name]; ok {
			projs = append(projs, projectItems{Name: proj.name, Items: items})
		}
	}

	pg := phaseGroup{Phase: phaseNum, Projects: projs}
	return renderPhaseDetail([]phaseGroup{pg}, phaseNum)
}

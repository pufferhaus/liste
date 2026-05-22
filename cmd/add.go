package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/mattn/go-isatty"
	"github.com/pufferhaus/liste/internal/model"
	"github.com/pufferhaus/liste/internal/store"
	"github.com/spf13/cobra"
)

var (
	addPriority string
	addTags     []string
	addStatus   string
	addPhase    int
)

var addCmd = &cobra.Command{
	Use:   "add <type> <title>",
	Short: "Create a new item",
	Long:  "Create a new item of the given type (feature, bug, idea, task, epic). Run without arguments in a terminal to use an interactive form.",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil // TTY-gated form handled in RunE
		}
		if len(args) < 2 {
			return fmt.Errorf("requires at least 2 arg(s) (<type> <title>), received %d", len(args))
		}
		return nil
	},
	RunE: runAdd,
}

func init() {
	addCmd.Flags().StringVar(&addPriority, "priority", "", "Priority (critical, high, medium, low)")
	addCmd.Flags().StringSliceVar(&addTags, "tag", nil, "Tags (can be specified multiple times)")
	addCmd.Flags().StringVar(&addStatus, "status", "", "Initial status (overrides default)")
	addCmd.Flags().IntVar(&addPhase, "phase", 0, "Phase number (0 = unphased)")
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		if !isatty.IsTerminal(os.Stdin.Fd()) {
			return fmt.Errorf("'liste add' requires <type> and <title> arguments when not running in a terminal")
		}
		return runAddInteractive(s)
	}

	typeStr := args[0]
	title := strings.Join(args[1:], " ")

	itemType, ok := model.ParseItemType(typeStr)
	if !ok {
		return fmt.Errorf("invalid type %q (valid: feature, bug, idea, task, epic)", typeStr)
	}

	cfg, err := s.ReadConfig()
	if err != nil {
		return err
	}

	item, err := s.CreateItem(itemType, title, cfg)
	if err != nil {
		return err
	}

	changed := false
	if addPriority != "" {
		if !cfg.IsValidPriority(addPriority) {
			return fmt.Errorf("invalid priority %q (valid: %s)", addPriority, strings.Join(cfg.Priorities, ", "))
		}
		item.Priority = addPriority
		changed = true
	}
	if addStatus != "" {
		if !cfg.IsValidStatus(addStatus) {
			return fmt.Errorf("invalid status %q (valid: %s)", addStatus, strings.Join(cfg.Statuses, ", "))
		}
		item.Status = addStatus
		changed = true
	}
	if len(addTags) > 0 {
		item.Tags = addTags
		changed = true
	}
	if addPhase > 0 {
		p := addPhase
		item.Phase = &p
		changed = true
	}
	if changed {
		item.Updated = time.Now()
		if err := s.WriteItem(item); err != nil {
			return err
		}
	}

	f := getFormatter()
	f.ItemCreated(item)
	return nil
}

func runAddInteractive(s *store.Store) error {
	cfg, err := s.ReadConfig()
	if err != nil {
		return err
	}

	var (
		itemType string = "feature"
		title    string
		priority string = cfg.Defaults.Priority
		phaseStr string
		tagsStr  string
	)

	priorityOpts := make([]huh.Option[string], len(cfg.Priorities))
	for i, p := range cfg.Priorities {
		label := strings.ToUpper(p[:1]) + p[1:]
		priorityOpts[i] = huh.NewOption(label, p)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Type").
				Options(
					huh.NewOption("Feature", "feature"),
					huh.NewOption("Bug", "bug"),
					huh.NewOption("Task", "task"),
					huh.NewOption("Idea", "idea"),
					huh.NewOption("Epic", "epic"),
				).
				Value(&itemType),
			huh.NewInput().
				Title("Title").
				Value(&title).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("title is required")
					}
					return nil
				}),
			huh.NewSelect[string]().
				Title("Priority").
				Options(priorityOpts...).
				Value(&priority),
			huh.NewInput().
				Title("Phase (optional, positive integer)").
				Value(&phaseStr),
			huh.NewInput().
				Title("Tags (optional, comma-separated)").
				Value(&tagsStr),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	t, _ := model.ParseItemType(itemType)
	item, err := s.CreateItem(t, strings.TrimSpace(title), cfg)
	if err != nil {
		return err
	}

	changed := false
	if priority != cfg.Defaults.Priority {
		item.Priority = priority
		changed = true
	}
	if phaseStr != "" {
		p, err := strconv.Atoi(strings.TrimSpace(phaseStr))
		if err != nil || p < 1 {
			return fmt.Errorf("phase must be a positive integer, got %q", phaseStr)
		}
		item.Phase = &p
		changed = true
	}
	if tagsStr != "" {
		var tags []string
		for _, tag := range strings.Split(tagsStr, ",") {
			if t := strings.TrimSpace(tag); t != "" {
				tags = append(tags, t)
			}
		}
		if len(tags) > 0 {
			item.Tags = tags
			changed = true
		}
	}
	if changed {
		item.Updated = time.Now()
		if err := s.WriteItem(item); err != nil {
			return err
		}
	}

	f := getFormatter()
	f.ItemCreated(item)
	return nil
}

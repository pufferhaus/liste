package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/mattn/go-isatty"
	"github.com/pufferhaus/liste/internal/store"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Initialize a new .liste/ in the current directory",
	Long:  "Creates a .liste/ directory with default config and state files. Run without arguments in a terminal for an interactive prompt.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	var name string
	switch {
	case len(args) > 0:
		name = args[0]
	case isatty.IsTerminal(os.Stdin.Fd()):
		name, err = promptProjectName(filepath.Base(cwd))
		if err != nil {
			return err
		}
	default:
		name = filepath.Base(cwd)
	}

	roadmapPath := filepath.Join(cwd, ".liste")
	s := store.New(roadmapPath)

	if s.Exists() {
		return fmt.Errorf(".liste/ already exists in %s", cwd)
	}

	if err := s.Init(name); err != nil {
		return err
	}

	f := getFormatter()
	f.Message(fmt.Sprintf("Initialized .liste/ for project %q", name))
	return nil
}

func promptProjectName(defaultName string) (string, error) {
	var name string = defaultName
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Project name").
				Value(&name).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("project name is required")
					}
					return nil
				}),
		),
	)
	if err := form.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(name), nil
}

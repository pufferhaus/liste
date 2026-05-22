package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pufferhaus/liste/internal/model"
	"gopkg.in/yaml.v3"
)

const (
	roadmapDir = ".liste"
	configFile = "config.yaml"
	stateFile  = ".state.yaml"
)

// Store manages reading and writing items in a .liste/ directory.
type Store struct {
	root string // absolute path to the .liste/ directory
}

// New creates a Store for the given .liste/ directory path.
func New(roadmapPath string) *Store {
	return &Store{root: roadmapPath}
}

// Root returns the store's root path.
func (s *Store) Root() string {
	return s.root
}

// Init creates the .liste/ directory with config and state files.
func (s *Store) Init(name string) error {
	if err := os.MkdirAll(s.root, 0755); err != nil {
		return fmt.Errorf("creating .liste directory: %w", err)
	}

	cfg := model.DefaultConfig(name)
	if err := s.WriteConfig(&cfg); err != nil {
		return err
	}

	state := model.DefaultState()
	if err := s.WriteState(&state); err != nil {
		return err
	}

	return nil
}

// ReadConfig reads the config.yaml file.
func (s *Store) ReadConfig() (*model.Config, error) {
	path := filepath.Join(s.root, configFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config not found at %s (run 'liste init' first)", path)
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("config file is empty at %s", path)
	}

	var cfg model.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config (file may be corrupted): %w", err)
	}

	// Validate minimum required fields
	if cfg.Project == "" {
		return nil, fmt.Errorf("config missing 'project' field at %s", path)
	}
	if len(cfg.Statuses) == 0 {
		cfg.Statuses = model.DefaultConfig("").Statuses
	}
	if cfg.Defaults.Status == "" {
		cfg.Defaults.Status = "idea"
	}
	if cfg.Defaults.Priority == "" {
		cfg.Defaults.Priority = "medium"
	}

	return &cfg, nil
}

// WriteConfig writes the config.yaml file.
func (s *Store) WriteConfig(cfg *model.Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(filepath.Join(s.root, configFile), data, 0644)
}

// ReadState reads the .state.yaml file.
func (s *Store) ReadState() (*model.State, error) {
	data, err := os.ReadFile(filepath.Join(s.root, stateFile))
	if err != nil {
		if os.IsNotExist(err) {
			state := model.DefaultState()
			return &state, nil
		}
		return nil, fmt.Errorf("reading state: %w", err)
	}

	if len(data) == 0 {
		// Empty state file — return defaults
		state := model.DefaultState()
		return &state, nil
	}

	var state model.State
	if err := yaml.Unmarshal(data, &state); err != nil {
		// Corrupted state file — reset to defaults and attempt repair
		state = model.DefaultState()
		_ = s.WriteState(&state) // best-effort repair
		return &state, nil
	}

	// Ensure NextIDs map is initialized
	if state.NextIDs == nil {
		state.NextIDs = model.DefaultState().NextIDs
	}

	return &state, nil
}

// WriteState writes the .state.yaml file.
func (s *Store) WriteState(state *model.State) error {
	data, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	return os.WriteFile(filepath.Join(s.root, stateFile), data, 0644)
}

// NextID allocates the next ID for the given item type and persists state.
func (s *Store) NextID(itemType model.ItemType) (string, error) {
	state, err := s.ReadState()
	if err != nil {
		return "", err
	}

	prefix := itemType.Prefix()
	num, ok := state.NextIDs[prefix]
	if !ok {
		num = 1
	}

	id := fmt.Sprintf("%s-%03d", prefix, num)
	state.NextIDs[prefix] = num + 1

	if err := s.WriteState(state); err != nil {
		return "", err
	}

	return id, nil
}

// WriteItem writes an item to its markdown file.
func (s *Store) WriteItem(item *model.Item) error {
	if item == nil {
		return fmt.Errorf("cannot write nil item")
	}
	if item.ID == "" {
		return fmt.Errorf("cannot write item with empty ID")
	}
	if item.Title == "" {
		return fmt.Errorf("cannot write item with empty title")
	}

	data, err := MarshalItem(item)
	if err != nil {
		return fmt.Errorf("marshaling item %s: %w", item.ID, err)
	}

	filename := item.ID + ".md"
	return os.WriteFile(filepath.Join(s.root, filename), data, 0644)
}

// ReadItem reads a single item by ID.
func (s *Store) ReadItem(id string) (*model.Item, error) {
	filename := id + ".md"
	data, err := os.ReadFile(filepath.Join(s.root, filename))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("item %s not found", id)
		}
		return nil, fmt.Errorf("reading item %s: %w", id, err)
	}

	return ParseItem(data)
}

// ListItems reads all items from the .liste/ directory.
func (s *Store) ListItems() ([]*model.Item, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, fmt.Errorf("reading roadmap directory: %w", err)
	}

	var items []*model.Item
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		// Skip files that don't look like item IDs
		if !isItemFilename(name) {
			continue
		}

		data, err := os.ReadFile(filepath.Join(s.root, name))
		if err != nil {
			continue // skip unreadable files
		}

		item, err := ParseItem(data)
		if err != nil {
			continue // skip unparseable files
		}

		items = append(items, item)
	}

	// Sort by priority weight, then by created date
	sort.Slice(items, func(i, j int) bool {
		pi := priorityWeight(items[i].Priority)
		pj := priorityWeight(items[j].Priority)
		if pi != pj {
			return pi < pj
		}
		return items[i].Created.Before(items[j].Created)
	})

	return items, nil
}

// DeleteItem removes an item file.
func (s *Store) DeleteItem(id string) error {
	filename := id + ".md"
	path := filepath.Join(s.root, filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("item %s not found", id)
	}
	return os.Remove(path)
}

// CreateItem creates a new item with a generated ID and writes it.
func (s *Store) CreateItem(itemType model.ItemType, title string, cfg *model.Config) (*model.Item, error) {
	id, err := s.NextID(itemType)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	item := &model.Item{
		ID:       id,
		Type:     itemType,
		Title:    title,
		Status:   cfg.Defaults.Status,
		Priority: cfg.Defaults.Priority,
		Created:  now,
		Updated:  now,
	}

	if err := s.WriteItem(item); err != nil {
		return nil, err
	}

	return item, nil
}

// Exists checks whether the .liste/ directory exists at the store path.
func (s *Store) Exists() bool {
	info, err := os.Stat(s.root)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// isItemFilename checks if a filename looks like a valid item file (PREFIX-NNN.md).
func isItemFilename(name string) bool {
	// Must end with .md
	base := strings.TrimSuffix(name, ".md")
	// Must contain a dash
	parts := strings.SplitN(base, "-", 2)
	if len(parts) != 2 {
		return false
	}
	// First part must be a known prefix
	prefix := parts[0]
	switch prefix {
	case "FEAT", "BUG", "IDEA", "TASK", "EPIC":
		return true
	default:
		return false
	}
}

// priorityWeight returns a numeric weight for sorting (lower = higher priority).
func priorityWeight(p string) int {
	switch p {
	case "critical":
		return 0
	case "high":
		return 1
	case "medium":
		return 2
	case "low":
		return 3
	default:
		return 4
	}
}

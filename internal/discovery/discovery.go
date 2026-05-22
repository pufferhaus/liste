package discovery

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pufferhaus/liste/internal/store"
)

const roadmapDir = ".liste"

// Result holds discovered project locations.
type Result struct {
	Root        string   // absolute path to root .liste/
	SubProjects []SubProject // nested .liste/ directories
}

// SubProject represents a discovered sub-project.
type SubProject struct {
	Name string // relative path from root parent (e.g., "transaction-service")
	Path string // absolute path to the .liste/ directory
}

// FindRoot walks up from startDir looking for a .liste/ directory.
// Returns the absolute path to the .liste/ directory, or empty string if not found.
func FindRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(dir, roadmapDir)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return "", nil
		}
		dir = parent
	}
}

// FindSubProjects recursively scans from rootParent for nested .liste/ directories.
// rootParent is the parent directory of the root .liste/ (i.e., the workspace root).
func FindSubProjects(rootParent string) ([]SubProject, error) {
	var subs []SubProject
	rootRoadmap := filepath.Join(rootParent, roadmapDir)

	err := filepath.WalkDir(rootParent, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}

		if !d.IsDir() {
			return nil
		}

		// Skip hidden directories other than .liste
		name := d.Name()
		if strings.HasPrefix(name, ".") && name != roadmapDir {
			return filepath.SkipDir
		}

		// Skip node_modules, bin, obj, etc.
		switch name {
		case "node_modules", "bin", "obj", "vendor", "packages":
			return filepath.SkipDir
		}

		// If this is a .liste/ directory that isn't the root one
		if name == roadmapDir && path != rootRoadmap {
			// The project name is the relative path of the parent directory
			parentDir := filepath.Dir(path)
			relPath, err := filepath.Rel(rootParent, parentDir)
			if err != nil {
				relPath = parentDir
			}

			subs = append(subs, SubProject{
				Name: filepath.ToSlash(relPath),
				Path: path,
			})

			return filepath.SkipDir // don't recurse into .liste/
		}

		return nil
	})

	return subs, err
}

// Discover performs full project discovery from the given start directory.
func Discover(startDir string) (*Result, error) {
	root, err := FindRoot(startDir)
	if err != nil {
		return nil, err
	}
	if root == "" {
		return nil, nil // no project found
	}

	rootParent := filepath.Dir(root)
	subs, err := FindSubProjects(rootParent)
	if err != nil {
		return nil, err
	}

	return &Result{
		Root:        root,
		SubProjects: subs,
	}, nil
}

// StoreForProject returns a Store for a named sub-project, or the root if name is empty.
func StoreForProject(result *Result, name string) *store.Store {
	if name == "" {
		return store.New(result.Root)
	}
	for _, sub := range result.SubProjects {
		if sub.Name == name {
			return store.New(sub.Path)
		}
	}
	return nil
}

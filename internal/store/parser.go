package store

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/pufferhaus/liste/internal/model"
	"gopkg.in/yaml.v3"
)

const frontmatterDelimiter = "---"

// frontmatter is the YAML structure written to/read from item files.
type frontmatter struct {
	ID       string         `yaml:"id"`
	Type     string         `yaml:"type"`
	Title    string         `yaml:"title"`
	Status   string         `yaml:"status"`
	Priority string         `yaml:"priority"`
	Phase    *int           `yaml:"phase,omitempty"`
	Created  string         `yaml:"created"`
	Updated  string         `yaml:"updated"`
	Tags     []string       `yaml:"tags,omitempty"`
	Links    []model.Link   `yaml:"links,omitempty"`
	Blocked  *model.Blocked `yaml:"blocked,omitempty"`
}

const dateFormat = "2006-01-02"

// ParseItem parses a markdown file with YAML frontmatter into an Item.
func ParseItem(data []byte) (*model.Item, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty file")
	}

	fm, body, err := splitFrontmatter(data)
	if err != nil {
		return nil, err
	}

	var meta frontmatter
	if err := yaml.Unmarshal(fm, &meta); err != nil {
		return nil, fmt.Errorf("parsing frontmatter: %w", err)
	}

	// Validate required fields
	if meta.ID == "" {
		return nil, fmt.Errorf("missing required field: id")
	}
	if meta.Type == "" {
		return nil, fmt.Errorf("missing required field: type")
	}
	if meta.Title == "" {
		return nil, fmt.Errorf("missing required field: title")
	}
	if meta.Status == "" {
		return nil, fmt.Errorf("missing required field: status")
	}

	created, err := time.Parse(dateFormat, meta.Created)
	if err != nil {
		created = time.Now() // fallback for corrupted dates
	}
	updated, err := time.Parse(dateFormat, meta.Updated)
	if err != nil {
		updated = created // fallback to created date
	}

	// Default priority if missing
	priority := meta.Priority
	if priority == "" {
		priority = "medium"
	}

	item := &model.Item{
		ID:       meta.ID,
		Type:     model.ItemType(meta.Type),
		Title:    meta.Title,
		Status:   meta.Status,
		Priority: priority,
		Phase:    meta.Phase,
		Created:  created,
		Updated:  updated,
		Tags:     meta.Tags,
		Links:    meta.Links,
		Blocked:  meta.Blocked,
		Body:     body,
	}

	return item, nil
}

// MarshalItem converts an Item back to markdown with YAML frontmatter.
func MarshalItem(item *model.Item) ([]byte, error) {
	meta := frontmatter{
		ID:       item.ID,
		Type:     string(item.Type),
		Title:    item.Title,
		Status:   item.Status,
		Priority: item.Priority,
		Phase:    item.Phase,
		Created:  item.Created.Format(dateFormat),
		Updated:  item.Updated.Format(dateFormat),
		Tags:     item.Tags,
		Links:    item.Links,
		Blocked:  item.Blocked,
	}

	yamlBytes, err := yaml.Marshal(&meta)
	if err != nil {
		return nil, fmt.Errorf("marshaling frontmatter: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString(frontmatterDelimiter)
	buf.WriteString("\n")
	buf.Write(yamlBytes)
	buf.WriteString(frontmatterDelimiter)
	buf.WriteString("\n")
	if item.Body != "" {
		buf.WriteString("\n")
		buf.WriteString(item.Body)
		if !strings.HasSuffix(item.Body, "\n") {
			buf.WriteString("\n")
		}
	}

	return buf.Bytes(), nil
}

// splitFrontmatter splits a file into YAML frontmatter bytes and markdown body.
func splitFrontmatter(data []byte) ([]byte, string, error) {
	content := string(data)
	content = strings.TrimSpace(content)

	if !strings.HasPrefix(content, frontmatterDelimiter) {
		return nil, "", fmt.Errorf("file does not start with frontmatter delimiter")
	}

	// Skip past first --- and newline
	rest := content[len(frontmatterDelimiter):]
	if len(rest) == 0 {
		return nil, "", fmt.Errorf("no closing frontmatter delimiter found")
	}
	// Trim the newline after opening delimiter
	if rest[0] == '\n' {
		rest = rest[1:]
	} else if rest[0] == '\r' && len(rest) > 1 && rest[1] == '\n' {
		rest = rest[2:]
	} else {
		return nil, "", fmt.Errorf("no newline after opening frontmatter delimiter")
	}

	// Find the closing delimiter (must be on its own line)
	idx := strings.Index(rest, "\n"+frontmatterDelimiter)
	if idx < 0 {
		// Check if the entire content ends with just the delimiter (no trailing newline)
		if strings.HasSuffix(rest, frontmatterDelimiter) && strings.Count(rest, frontmatterDelimiter) == 1 {
			idx = len(rest) - len(frontmatterDelimiter) - 1
			if idx < 0 {
				return nil, "", fmt.Errorf("no closing frontmatter delimiter found")
			}
		} else {
			return nil, "", fmt.Errorf("no closing frontmatter delimiter found")
		}
	}

	fmContent := rest[:idx]
	// Calculate where body starts (after closing delimiter + optional newline)
	afterIdx := idx + 1 + len(frontmatterDelimiter)
	body := ""
	if afterIdx < len(rest) {
		body = rest[afterIdx:]
		body = strings.TrimLeft(body, "\r\n")
	}

	// Validate frontmatter is not empty
	if strings.TrimSpace(fmContent) == "" {
		return nil, "", fmt.Errorf("frontmatter is empty")
	}

	return []byte(fmContent), body, nil
}

package projectcache

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const DefaultLimit = 20

type Cache struct {
	path  string
	limit int
	mu    sync.Mutex
}

type entry struct {
	Path       string    `json:"path"`
	LastUsedAt time.Time `json:"lastUsedAt"`
}

type diskState struct {
	Projects []entry `json:"projects"`
}

func New(path string, limit int) *Cache {
	if limit <= 0 {
		limit = DefaultLimit
	}
	return &Cache{
		path:  strings.TrimSpace(path),
		limit: limit,
	}
}

func (c *Cache) List() ([]string, error) {
	if c == nil || c.path == "" {
		return []string{}, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	state, err := c.read()
	if err != nil {
		return nil, err
	}

	projects := make([]string, 0, len(state.Projects))
	for _, project := range state.Projects {
		if project.Path == "" {
			continue
		}
		projects = append(projects, project.Path)
	}
	return projects, nil
}

func (c *Cache) Add(path string) error {
	if c == nil || c.path == "" {
		return nil
	}

	path = normalizePath(path)
	if path == "" {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	state, err := c.read()
	if err != nil {
		return err
	}

	updated := []entry{{Path: path, LastUsedAt: time.Now().UTC()}}
	for _, project := range state.Projects {
		if project.Path == "" || project.Path == path {
			continue
		}
		updated = append(updated, project)
		if len(updated) >= c.limit {
			break
		}
	}
	state.Projects = updated
	return c.write(state)
}

func (c *Cache) read() (diskState, error) {
	data, err := os.ReadFile(c.path)
	if errors.Is(err, os.ErrNotExist) {
		return diskState{}, nil
	}
	if err != nil {
		return diskState{}, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return diskState{}, nil
	}

	var state diskState
	if err := json.Unmarshal(data, &state); err != nil {
		return diskState{}, err
	}
	if len(state.Projects) > c.limit {
		state.Projects = state.Projects[:c.limit]
	}
	return state, nil
}

func (c *Cache) write(state diskState) error {
	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(filepath.Dir(c.path), ".projects-*.json")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, c.path)
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}

// Package cache provides a JSON-file persistence layer.
package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/jelloeater-agent/modelhub/internal/model"
)

// Store manages cached model data.
type Store struct {
	path string // resolved absolute path to cache file
}

// NewStore creates a cache store at the given path (supports ~/ expansion).
func NewStore(path string) (*Store, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, path[2:])
	}
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &Store{path: path}, nil
}

// Load reads cached data. Returns nil if no cache exists (first run).
func (s *Store) Load() (*model.Cache, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // First run, no cache yet
		}
		return nil, err
	}
	var c model.Cache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// Save persists model data and fetch timestamps.
func (s *Store) Save(c *model.Cache) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

// Exists checks if a cache file exists.
func (s *Store) Exists() bool {
	_, err := os.Stat(s.path)
	return err == nil
}

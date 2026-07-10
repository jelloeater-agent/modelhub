package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jelloeater-agent/modelhub/internal/model"
)

func TestNewStore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if s.path != path {
		t.Errorf("path = %q, want %q", s.path, path)
	}
}

func TestNewStoreCreatesDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "cache.json")

	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(s.path)); os.IsNotExist(err) {
		t.Error("directory was not created")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")
	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	c := &model.Cache{
		Models: []model.Model{
			{ID: "test/model", Name: "Test Model", Provider: "Test", Sources: []string{"bifrost"}},
		},
		FetchedAt: map[string]string{"bifrost": "2025-01-15T10:00:00Z"},
		Version:   1,
	}

	if err := s.Save(c); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded.Models) != 1 {
		t.Fatalf("loaded %d models, want 1", len(loaded.Models))
	}
	if loaded.Models[0].ID != "test/model" {
		t.Errorf("ID = %q, want test/model", loaded.Models[0].ID)
	}
	if loaded.FetchedAt["bifrost"] != "2025-01-15T10:00:00Z" {
		t.Errorf("FetchedAt = %q", loaded.FetchedAt["bifrost"])
	}
	if loaded.Version != 1 {
		t.Errorf("Version = %d, want 1", loaded.Version)
	}
}

func TestLoadFirstRun(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")
	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	c, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c != nil {
		t.Error("expected nil for first run (no cache file)")
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")
	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	if s.Exists() {
		t.Error("Exists() should be false before save")
	}

	s.Save(&model.Cache{Models: []model.Model{{ID: "a"}}})

	if !s.Exists() {
		t.Error("Exists() should be true after save")
	}
}

func TestLoadCorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")
	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	// Write invalid JSON
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = s.Load()
	if err == nil {
		t.Error("expected error for corrupt cache file")
	}
}

func TestNewStoreHomeExpansion(t *testing.T) {
	// Just verify it doesn't crash and returns a valid path
	s, err := NewStore("~/modelhub-test/cache.json")
	if err != nil {
		t.Fatalf("NewStore with ~/: %v", err)
	}
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, "modelhub-test", "cache.json")
	if s.path != expected {
		t.Errorf("path = %q, want %q", s.path, expected)
	}
	// Cleanup
	os.RemoveAll(filepath.Join(home, "modelhub-test"))
}

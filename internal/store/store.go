// Package store implements the local-first persistence layer for wwlog.
// Each day's log is stored as a JSON file named YYYY-MM-DD.json under a
// user-configurable directory (default: alongside the config file).
// The directory can be placed in any cloud-synced folder (Dropbox, iCloud
// Drive, etc.) for multi-device access.
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ali5ter/wwlog/internal/api"
)

// Store is a directory-backed day-log cache.
type Store struct {
	dir string
}

// New opens (or creates) a store rooted at dir.
func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

// Dir returns the store's root directory.
func (s *Store) Dir() string { return s.dir }

// Load reads a single day log from the store. Returns false if the date is
// not present or the file cannot be decoded.
func (s *Store) Load(date string) (*api.DayLog, bool) {
	data, err := os.ReadFile(filepath.Join(s.dir, date+".json"))
	if err != nil {
		return nil, false
	}
	var day api.DayLog
	if err := json.Unmarshal(data, &day); err != nil {
		return nil, false
	}
	return &day, true
}

// Save writes a day log to the store, overwriting any existing file for
// that date. Errors are non-fatal — the caller decides whether to surface
// them.
func (s *Store) Save(day *api.DayLog) error {
	data, err := json.Marshal(day)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dir, day.Date+".json"), data, 0o644)
}

// Dates returns all dates present in the store, sorted ascending.
func (s *Store) Dates() []string {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil
	}
	var dates []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			dates = append(dates, strings.TrimSuffix(e.Name(), ".json"))
		}
	}
	sort.Strings(dates)
	return dates
}

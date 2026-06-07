package repl

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxHistorySize = 1000
	historyFile    = "history"
)

// History manages query history.
type History struct {
	entries []string
	path    string
}

// NewHistory creates a new history manager.
func NewHistory() *History {
	return &History{
		path: historyPath(),
	}
}

// historyPath returns the path to the history file.
func historyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ahastudio", historyFile)
}

// Load loads history from disk.
func (h *History) Load() error {
	if h.path == "" {
		return nil
	}

	file, err := os.Open(h.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			h.entries = append(h.entries, line)
		}
	}

	// Keep only recent entries
	if len(h.entries) > maxHistorySize {
		h.entries = h.entries[len(h.entries)-maxHistorySize:]
	}

	return scanner.Err()
}

// Save saves history to disk.
func (h *History) Save() error {
	if h.path == "" {
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(h.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	file, err := os.Create(h.path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	// Keep only recent entries
	entries := h.entries
	if len(entries) > maxHistorySize {
		entries = entries[len(entries)-maxHistorySize:]
	}

	for _, entry := range entries {
		if _, err := file.WriteString(entry + "\n"); err != nil {
			return err
		}
	}

	return nil
}

// Add adds an entry to history.
func (h *History) Add(entry string) {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return
	}

	// Don't add duplicates of the last entry
	if len(h.entries) > 0 && h.entries[len(h.entries)-1] == entry {
		return
	}

	h.entries = append(h.entries, entry)
}

// Entries returns all history entries.
func (h *History) Entries() []string {
	return h.entries
}

// Clear clears all history.
func (h *History) Clear() {
	h.entries = nil
}

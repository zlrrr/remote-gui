package record

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Record represents a single script execution record.
type Record struct {
	ID         string            `json:"record_id"`
	Script     string            `json:"script"`
	Params     map[string]string `json:"params"`
	Status     string            `json:"status"`
	ExitCode   int               `json:"exit_code"`
	Stdout     string            `json:"stdout"`
	Stderr     string            `json:"stderr"`
	DurationMs int64             `json:"duration_ms"`
	ExecutedAt time.Time         `json:"executed_at"`
}

// ListResult is the result of listing records.
type ListResult struct {
	Total    int
	Page     int
	PageSize int
	Records  []*Record
}

// Store persists and retrieves execution records.
type Store interface {
	Save(rec Record) (string, error)
	Get(id string) (*Record, error)
	List(page, pageSize int) (*ListResult, error)
}

// NewFileStore creates a Store backed by individual JSON files in the given directory.
func NewFileStore(dir string) Store {
	return &fileStore{dir: dir}
}

type fileStore struct {
	dir string
	mu  sync.RWMutex
}

// generateID produces a unique record ID like rec-20250301-143022-abc123.
func generateID(t time.Time) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	suffix := make([]byte, 6)
	for i := range suffix {
		suffix[i] = letters[rand.Intn(len(letters))]
	}
	return fmt.Sprintf("rec-%s-%s", t.UTC().Format("20060102-150405"), string(suffix))
}

func (s *fileStore) Save(rec Record) (string, error) {
	if err := os.MkdirAll(s.dir, 0o750); err != nil {
		return "", fmt.Errorf("cannot create records directory: %w", err)
	}

	id := generateID(rec.ExecutedAt)
	rec.ID = id

	data, err := json.Marshal(rec)
	if err != nil {
		return "", fmt.Errorf("failed to marshal record: %w", err)
	}

	path := filepath.Join(s.dir, id+".json")

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.WriteFile(path, data, 0o640); err != nil {
		return "", fmt.Errorf("failed to write record file: %w", err)
	}

	return id, nil
}

func (s *fileStore) Get(id string) (*Record, error) {
	path := filepath.Join(s.dir, id+".json")

	s.mu.RLock()
	data, err := os.ReadFile(path)
	s.mu.RUnlock()

	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("record %q not found", id)
		}
		return nil, fmt.Errorf("failed to read record file: %w", err)
	}

	var rec Record
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, fmt.Errorf("failed to parse record: %w", err)
	}

	return &rec, nil
}

func (s *fileStore) List(page, pageSize int) (*ListResult, error) {
	s.mu.RLock()
	entries, err := os.ReadDir(s.dir)
	s.mu.RUnlock()

	if err != nil {
		if os.IsNotExist(err) {
			return &ListResult{Page: page, PageSize: pageSize, Records: []*Record{}}, nil
		}
		return nil, fmt.Errorf("failed to read records directory: %w", err)
	}

	// Collect record filenames (only .json files)
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}

	// Sort by filename descending (newest first, IDs embed timestamp)
	sort.Sort(sort.Reverse(sort.StringSlice(names)))

	total := len(names)

	// Paginate
	start := (page - 1) * pageSize
	if start >= total {
		return &ListResult{Total: total, Page: page, PageSize: pageSize, Records: []*Record{}}, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	pageNames := names[start:end]

	records := make([]*Record, 0, len(pageNames))
	for _, name := range pageNames {
		id := strings.TrimSuffix(name, ".json")
		rec, err := s.Get(id)
		if err != nil {
			continue // skip corrupted records
		}
		records = append(records, rec)
	}

	return &ListResult{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Records:  records,
	}, nil
}

package record

import "time"

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

// NewFileStore creates a Store backed by JSON Lines files in the given directory.
func NewFileStore(dir string) Store {
	return &fileStore{dir: dir}
}

type fileStore struct {
	dir string
}

func (s *fileStore) Save(rec Record) (string, error) {
	// TODO: implement in Phase 2.2
	return "", nil
}

func (s *fileStore) Get(id string) (*Record, error) {
	// TODO: implement in Phase 2.2
	return nil, nil
}

func (s *fileStore) List(page, pageSize int) (*ListResult, error) {
	// TODO: implement in Phase 2.2
	return nil, nil
}

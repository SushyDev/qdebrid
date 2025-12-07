package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Record represents a tracked torrent
type Record struct {
	Hash         string    `json:"hash"`         // The torrent hash/ID we return to Servarr
	DownloadID   string    `json:"downloadId"`   // Servarr's download ID (usually same as Hash)
	RealDebridID string    `json:"realDebridId"` // Real-Debrid's internal ID
	AddedAt      time.Time `json:"addedAt"`      // When this was added
	LastSeenAt   time.Time `json:"lastSeenAt"`   // Last time we saw this torrent
	ServarrHost  string    `json:"servarrHost"`  // Which Servarr instance added this
	Status       string    `json:"status"`       // Current status
	Title        string    `json:"title"`        // Torrent title/name
}

// Store manages persistent history of tracked torrents
type Store struct {
	mu       sync.RWMutex
	records  map[string]*Record // keyed by hash
	filePath string
	logger   *zap.Logger
}

// NewStore creates a new history store
func NewStore(dataDir string, logger *zap.Logger) (*Store, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	filePath := filepath.Join(dataDir, "history.json")

	store := &Store{
		records:  make(map[string]*Record),
		filePath: filePath,
		logger:   logger,
	}

	// Load existing history
	if err := store.load(); err != nil {
		logger.Warn("failed to load history, starting fresh", zap.Error(err))
	}

	return store, nil
}

// Add adds or updates a torrent record
func (s *Store) Add(record *Record) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record.LastSeenAt = time.Now()

	// If this is a new record, set AddedAt
	if existing, ok := s.records[record.Hash]; !ok || existing.AddedAt.IsZero() {
		record.AddedAt = time.Now()
	} else {
		// Preserve original AddedAt
		record.AddedAt = existing.AddedAt
	}

	s.records[record.Hash] = record
}

// Get retrieves a record by hash
func (s *Store) Get(hash string) (*Record, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, ok := s.records[hash]
	return record, ok
}

// GetByDownloadID retrieves a record by Servarr's downloadId
func (s *Store) GetByDownloadID(downloadID string) (*Record, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, record := range s.records {
		if record.DownloadID == downloadID {
			return record, true
		}
	}
	return nil, false
}

// GetByRealDebridID retrieves a record by Real-Debrid's ID
func (s *Store) GetByRealDebridID(rdID string) (*Record, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, record := range s.records {
		if record.RealDebridID == rdID {
			return record, true
		}
	}
	return nil, false
}

// GetAll returns all records
func (s *Store) GetAll() []*Record {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := make([]*Record, 0, len(s.records))
	for _, record := range s.records {
		records = append(records, record)
	}
	return records
}

// GetHashes returns all tracked hashes
func (s *Store) GetHashes() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hashes := make([]string, 0, len(s.records))
	for hash := range s.records {
		hashes = append(hashes, hash)
	}
	return hashes
}

// Remove removes a record by hash
func (s *Store) Remove(hash string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.records, hash)
}

// Cleanup removes old records that haven't been seen recently
func (s *Store) Cleanup(maxAge time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for hash, record := range s.records {
		if record.LastSeenAt.Before(cutoff) {
			delete(s.records, hash)
			removed++
		}
	}

	return removed
}

// Save persists the history to disk
func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Convert map to slice for JSON
	records := make([]*Record, 0, len(s.records))
	for _, record := range s.records {
		records = append(records, record)
	}

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal history: %w", err)
	}

	// Write to temp file first, then rename (atomic operation)
	tempFile := s.filePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := os.Rename(tempFile, s.filePath); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}

// load reads history from disk
func (s *Store) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No history file yet, that's ok
		}
		return fmt.Errorf("read history file: %w", err)
	}

	var records []*Record
	if err := json.Unmarshal(data, &records); err != nil {
		return fmt.Errorf("unmarshal history: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Convert slice to map
	for _, record := range records {
		s.records[record.Hash] = record
	}

	s.logger.Info("loaded history", zap.Int("records", len(records)))
	return nil
}

// Count returns the number of tracked torrents
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.records)
}

package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"max-bot/joomla"
)

// CallbackResult represents the result of a callback button press
type CallbackResult struct {
	CallbackID   string            `json:"callback_id"`
	UserID       int64             `json:"user_id"`
	ChatID       int64             `json:"chat_id"`
	Payload      string            `json:"payload"`
	MessageText  string            `json:"message_text"`
	ProcessedAt  time.Time         `json:"processed_at"`
	Analysis     *joomla.AnalysisResult `json:"analysis,omitempty"`
	PlannedChanges []joomla.Change `json:"planned_changes,omitempty"`
}

// CallbackStorage handles persistent storage of callback results
type CallbackStorage struct {
	filePath string
	results  map[string]CallbackResult
	mutex    sync.RWMutex
}

// NewCallbackStorage creates a new callback storage
func NewCallbackStorage(filePath string) (*CallbackStorage, error) {
	storage := &CallbackStorage{
		filePath: filePath,
		results:  make(map[string]CallbackResult),
	}

	// Load existing data from file
	if err := storage.Load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load callback storage: %w", err)
		}
		// File doesn't exist, that's ok - we'll create it on first save
	}

	return storage, nil
}

// Load reads callback results from the JSON file
func (s *CallbackStorage) Load() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var results map[string]CallbackResult
	if err := json.Unmarshal(data, &results); err != nil {
		return fmt.Errorf("failed to unmarshal callback results: %w", err)
	}

	s.results = results
	return nil
}

// Save writes callback results to the JSON file
func (s *CallbackStorage) Save() error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	data, err := json.MarshalIndent(s.results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal callback results: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write callback results: %w", err)
	}

	return nil
}

// GetResult retrieves a callback result by ID
func (s *CallbackStorage) GetResult(callbackID string) (CallbackResult, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result, exists := s.results[callbackID]
	return result, exists
}

// AddResult adds a new callback result
func (s *CallbackStorage) AddResult(result CallbackResult) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.results[result.CallbackID] = result

	// Save to file after adding
	return s.Save()
}

// Exists checks if a callback has already been processed
func (s *CallbackStorage) Exists(callbackID string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	_, exists := s.results[callbackID]
	return exists
}

// GetStats returns statistics about processed callbacks
func (s *CallbackStorage) GetStats() (total int, accepted int, cancelled int) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	total = len(s.results)
	for _, result := range s.results {
		switch result.Payload {
		case "accept":
			accepted++
		case "cancel":
			cancelled++
		}
	}
	return
}

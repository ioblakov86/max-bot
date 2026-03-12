package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"max-bot/joomla"
)

// PendingConfirmation represents a pending user confirmation
type PendingConfirmation struct {
	MessageID      string                `json:"message_id"`       // ID сообщения бота
	ParentMessageID string               `json:"parent_message_id"` // ID исходного сообщения (на которое был ответ)
	ChatID         int64                 `json:"chat_id"`
	UserID         int64                 `json:"user_id"` // ID пользователя, который должен подтвердить
	Analysis       joomla.AnalysisResult `json:"analysis"`
	PlannedChanges []joomla.Change       `json:"planned_changes"`
	CreatedAt      time.Time             `json:"created_at"`
	Answered       bool                  `json:"answered"` // Был ли уже получен ответ
	Answer         string                `json:"answer,omitempty"` // Полученный ответ
}

// ConfirmationStorage handles persistent storage of pending confirmations
type ConfirmationStorage struct {
	filePath      string
	confirmations map[string]*PendingConfirmation // key: message_id
	Mutex         sync.RWMutex
}

// NewConfirmationStorage creates a new confirmation storage
func NewConfirmationStorage(filePath string) (*ConfirmationStorage, error) {
	storage := &ConfirmationStorage{
		filePath:      filePath,
		confirmations: make(map[string]*PendingConfirmation),
	}

	// Load existing data from file
	if err := storage.Load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load confirmation storage: %w", err)
		}
	}

	return storage, nil
}

// Load reads confirmation data from the JSON file
func (s *ConfirmationStorage) Load() error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var confirmations map[string]*PendingConfirmation
	if err := json.Unmarshal(data, &confirmations); err != nil {
		return fmt.Errorf("failed to unmarshal confirmations: %w", err)
	}

	s.confirmations = confirmations
	return nil
}

// Save writes confirmation data to the JSON file
func (s *ConfirmationStorage) Save() error {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()

	data, err := json.MarshalIndent(s.confirmations, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal confirmations: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write confirmations: %w", err)
	}

	return nil
}

// AddConfirmation adds a new pending confirmation
func (s *ConfirmationStorage) AddConfirmation(conf *PendingConfirmation) error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	s.confirmations[conf.MessageID] = conf
	return s.Save()
}

// GetConfirmation retrieves a confirmation by message ID
func (s *ConfirmationStorage) GetConfirmation(messageID string) (*PendingConfirmation, bool) {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()

	conf, exists := s.confirmations[messageID]
	return conf, exists
}

// MarkAnswered marks a confirmation as answered
func (s *ConfirmationStorage) MarkAnswered(messageID string, answer string) error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	if conf, exists := s.confirmations[messageID]; exists {
		conf.Answered = true
		conf.Answer = answer
		return s.Save()
	}
	return fmt.Errorf("confirmation not found")
}

// GetPendingForUser gets the first pending (not answered) confirmation for a user
func (s *ConfirmationStorage) GetPendingForUser(userID int64) (*PendingConfirmation, string) {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()
	
	for msgID, conf := range s.confirmations {
		if !conf.Answered && conf.UserID == userID {
			return conf, msgID
		}
	}
	return nil, ""
}

// IsAnswered checks if a confirmation has already been answered
func (s *ConfirmationStorage) IsAnswered(messageID string) bool {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()

	if conf, exists := s.confirmations[messageID]; exists {
		return conf.Answered
	}
	return false
}

// GetPendingForMessage gets a pending (not answered) confirmation for a message
func (s *ConfirmationStorage) GetPendingForMessage(messageID string) (*PendingConfirmation, bool) {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()

	if conf, exists := s.confirmations[messageID]; exists && !conf.Answered {
		return conf, true
	}
	return nil, false
}

// CleanupOld removes confirmations older than specified duration
func (s *ConfirmationStorage) CleanupOld(maxAge time.Duration) error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	now := time.Now()
	changed := false

	for id, conf := range s.confirmations {
		if now.Sub(conf.CreatedAt) > maxAge {
			delete(s.confirmations, id)
			changed = true
		}
	}

	if changed {
		return s.Save()
	}
	return nil
}

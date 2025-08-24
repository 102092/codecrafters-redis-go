package store

import (
	"time"
)

// ValueWithTTL represents a value with an expiration time
type ValueWithTTL struct {
	Value    string
	ExpireAt time.Time
}

// Store manages key-value storage with optional TTL support
type Store struct {
	storage       map[string]string       // Regular key-value storage
	expireStorage map[string]ValueWithTTL // Storage with TTL
}

// NewStore creates a new Store instance
func NewStore() *Store {
	return &Store{
		storage:       make(map[string]string),
		expireStorage: make(map[string]ValueWithTTL),
	}
}

// SET implements Redis SET command
// Supports both regular SET and SET with PX (milliseconds expiry)
func (s *Store) SET(key, value string, px *int) { // TODO handle different time unit
	if px != nil {
		// SET with expiry
		expireAt := time.Now().Add(time.Duration(*px) * time.Millisecond)
		s.expireStorage[key] = ValueWithTTL{
			Value:    value,
			ExpireAt: expireAt,
		}
		// Remove from regular storage if exists
		delete(s.storage, key)
	} else {
		// Regular SET without expiry
		s.storage[key] = value
		// Remove from expire storage if exists
		delete(s.expireStorage, key)
	}
}

// GET implements Redis GET command
// Returns nil if key doesn't exist or has expired
func (s *Store) GET(key string) *string {
	// Check expire storage first
	if obj, exists := s.expireStorage[key]; exists {
		now := time.Now()
		if obj.ExpireAt.Before(now) {
			// Key has expired, delete it
			delete(s.expireStorage, key)
			return nil
		}
		return &obj.Value
	}

	// Check regular storage
	if value, exists := s.storage[key]; exists {
		return &value
	}

	// Key not found
	return nil
}

package security

import (
	"context"
	"sync"
	"time"
)

// OwnedRecord is a value associated with the subject that created it.
type OwnedRecord[T any] struct {
	ID        string
	Owner     string
	CreatedAt time.Time
	Value     T
}

// Store is an in-memory subject-isolated record store.
type Store[T any] struct {
	mu      sync.RWMutex
	records map[string]OwnedRecord[T]
}

// Put stores value under id for the current subject.
func (s *Store[T]) Put(ctx context.Context, id string, value T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.records == nil {
		s.records = make(map[string]OwnedRecord[T])
	}
	s.records[id] = OwnedRecord[T]{
		ID:        id,
		Owner:     SubjectFromContext(ctx),
		CreatedAt: time.Now().UTC(),
		Value:     value,
	}
}

// Get returns a value only when it belongs to the current subject.
func (s *Store[T]) Get(ctx context.Context, id string) (T, bool, error) {
	s.mu.RLock()
	record, ok := s.records[id]
	s.mu.RUnlock()
	var zero T
	if !ok {
		return zero, false, nil
	}
	if err := RequireSubject(ctx, record.Owner); err != nil {
		return zero, false, err
	}
	return record.Value, true, nil
}

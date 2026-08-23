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

type storeKey struct {
	owner string
	id    string
}

// Store is an in-memory subject-isolated record store.
type Store[T any] struct {
	mu      sync.RWMutex
	records map[storeKey]OwnedRecord[T]
}

// Put stores value under id for the current subject.
func (s *Store[T]) Put(ctx context.Context, id string, value T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.records == nil {
		s.records = make(map[storeKey]OwnedRecord[T])
	}
	owner := SubjectFromContext(ctx)
	s.records[storeKey{owner: owner, id: id}] = OwnedRecord[T]{
		ID:        id,
		Owner:     owner,
		CreatedAt: time.Now().UTC(),
		Value:     value,
	}
}

// Get returns a value only when it belongs to the current subject.
func (s *Store[T]) Get(ctx context.Context, id string) (T, bool, error) {
	owner := SubjectFromContext(ctx)
	s.mu.RLock()
	record, ok := s.records[storeKey{owner: owner, id: id}]
	s.mu.RUnlock()
	var zero T
	if !ok {
		return zero, false, nil
	}
	return record.Value, true, nil
}

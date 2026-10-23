package security

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestStoreEvictsAndExpires(t *testing.T) {
	var store Store[string]
	for i := range maxStoredRecords + 5 {
		store.Put(t.Context(), fmt.Sprint(i), "value")
	}
	require.Len(t, store.records, maxStoredRecords)
	_, ok, err := store.Get(t.Context(), "0")
	require.NoError(t, err)
	require.False(t, ok)
	store.Put(t.Context(), "latest", "value")
	key := storeKey{owner: SubjectFromContext(t.Context()), id: "latest"}
	record := store.records[key]
	record.CreatedAt = time.Now().Add(-recordTTL - time.Second)
	store.records[key] = record
	_, ok, err = store.Get(t.Context(), "latest")
	require.NoError(t, err)
	require.False(t, ok)
	store.Put(t.Context(), "new", "value")
	_, exists := store.records[key]
	require.False(t, exists)
}

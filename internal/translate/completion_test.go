package translate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompleteBatchIDsReturnsLatestBatchForEmptyOrMatchingPrefix(t *testing.T) {
	svc := &Service{}
	svc.storeLatest(t.Context(), Batch{BatchID: "batch-fr-common-123"})

	require.Equal(t, []string{"batch-fr-common-123"}, svc.CompleteBatchIDs(t.Context(), ""))
	require.Equal(t, []string{"batch-fr-common-123"}, svc.CompleteBatchIDs(t.Context(), "batch-fr"))
}

func TestCompleteBatchIDsReturnsNilWithoutUsableLatestBatch(t *testing.T) {
	svc := &Service{}

	require.Nil(t, svc.CompleteBatchIDs(t.Context(), ""))

	svc.storeLatest(t.Context(), Batch{})
	require.Nil(t, svc.CompleteBatchIDs(t.Context(), ""))
}

func TestCompleteBatchIDsReturnsNilForNonMatchingPrefix(t *testing.T) {
	svc := &Service{}
	svc.storeLatest(t.Context(), Batch{BatchID: "batch-fr-common-123"})

	require.Nil(t, svc.CompleteBatchIDs(t.Context(), "batch-es"))
}

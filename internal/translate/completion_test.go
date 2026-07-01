package translate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompleteBatchIDsReturnsLatestBatchForEmptyOrMatchingPrefix(t *testing.T) {
	svc := &Service{}
	svc.storeLatest(Batch{BatchID: "batch-fr-common-123"})

	require.Equal(t, []string{"batch-fr-common-123"}, svc.CompleteBatchIDs(""))
	require.Equal(t, []string{"batch-fr-common-123"}, svc.CompleteBatchIDs("batch-fr"))
}

func TestCompleteBatchIDsReturnsNilWithoutUsableLatestBatch(t *testing.T) {
	svc := &Service{}

	require.Nil(t, svc.CompleteBatchIDs(""))

	svc.storeLatest(Batch{})
	require.Nil(t, svc.CompleteBatchIDs(""))
}

func TestCompleteBatchIDsReturnsNilForNonMatchingPrefix(t *testing.T) {
	svc := &Service{}
	svc.storeLatest(Batch{BatchID: "batch-fr-common-123"})

	require.Nil(t, svc.CompleteBatchIDs("batch-es"))
}

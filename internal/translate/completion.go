package translate

import (
	"context"
	"strings"
)

// CompleteBatchIDs returns matching known translation batch IDs for the current subject.
func (s *Service) CompleteBatchIDs(ctx context.Context, prefix string) []string {
	batch, ok, err := s.LatestPlan(ctx)
	if err != nil || !ok || batch.BatchID == "" {
		return nil
	}
	if prefix == "" || strings.HasPrefix(batch.BatchID, prefix) {
		return []string{batch.BatchID}
	}
	return nil
}

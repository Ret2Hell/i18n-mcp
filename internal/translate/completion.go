package translate

import "strings"

// CompleteBatchIDs returns matching known translation batch IDs.
func (s *Service) CompleteBatchIDs(prefix string) []string {
	batch, ok := s.LatestPlan()
	if !ok || batch.BatchID == "" {
		return nil
	}
	if prefix == "" || strings.HasPrefix(batch.BatchID, prefix) {
		return []string{batch.BatchID}
	}
	return nil
}

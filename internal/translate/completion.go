package translate

import "strings"

// CompleteBatchIDs returns known translation batch IDs matching prefix.
func (s *Service) CompleteBatchIDs(prefix string) []string {
	batch, ok := s.LatestPlan()
	if !ok || batch.BatchID == "" {
		return nil
	}
	prefix = strings.TrimSpace(prefix)
	if prefix != "" && !strings.HasPrefix(batch.BatchID, prefix) {
		return nil
	}
	return []string{batch.BatchID}
}

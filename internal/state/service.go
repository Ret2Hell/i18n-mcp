package state

import "context"

// Save persists the translation state file.
func (s *Service) Save(ctx context.Context, file File) error {
	return s.store.Save(ctx, file)
}

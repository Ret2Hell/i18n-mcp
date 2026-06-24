package state

import "context"

func (s *Service) Save(ctx context.Context, file File) error {
	return s.store.Save(ctx, file)
}

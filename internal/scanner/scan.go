package scanner

import (
	"context"
	"time"
)

func (s *Service) Scan(ctx context.Context, in ScanInput) (Report, error) {
	files, err := s.DiscoverSourceFiles(ctx, in.Files)
	if err != nil {
		return Report{}, err
	}
	report := Report{FilesScanned: len(files), Files: files, GeneratedAt: time.Now().UTC()}
	s.storeLatest(report)
	return report, nil
}

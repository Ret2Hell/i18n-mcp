package scanner

import (
	"context"
	"os"
	"sort"
	"time"
)

func (s *Service) Scan(ctx context.Context, in ScanInput) (Report, error) {
	files, err := s.DiscoverSourceFiles(ctx, in.Files)
	if err != nil {
		return Report{}, err
	}
	report := Report{FilesScanned: len(files), Files: files, GeneratedAt: time.Now().UTC()}
	usageByID := map[string]*Usage{}
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return Report{}, err
		}
		data, err := s.readSourceFile(file.Path)
		if err != nil {
			return Report{}, err
		}
		for _, ev := range scanLiteralCalls(file.Path, data) {
			addEvidence(usageByID, ev, "literal_call")
		}
		for _, ev := range scanJSXI18nKeys(file.Path, data) {
			addEvidence(usageByID, ev, "jsx_i18n_key")
		}
	}
	report.Usages = sortedUsages(usageByID)
	s.storeLatest(report)
	return report, nil
}

func (s *Service) readSourceFile(relPath string) ([]byte, error) {
	absPath, err := s.guard.ResolveExisting(relPath)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(absPath)
}

func addEvidence(usageByID map[string]*Usage, ev Evidence, kind string) {
	id := UsageIdentity(ev.Namespace, ev.Key)
	usage := usageByID[id]
	if usage == nil {
		usage = &Usage{Namespace: ev.Namespace, Key: ev.Key, FullKey: ev.FullKey, Kind: kind}
		usageByID[id] = usage
	}
	usage.Evidence = append(usage.Evidence, ev)
}

func sortedUsages(usageByID map[string]*Usage) []Usage {
	ids := make([]string, 0, len(usageByID))
	for id := range usageByID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]Usage, 0, len(ids))
	for _, id := range ids {
		usage := *usageByID[id]
		sort.Slice(usage.Evidence, func(i, j int) bool {
			return lessEvidence(usage.Evidence[i], usage.Evidence[j])
		})
		out = append(out, usage)
	}
	return out
}

func lessEvidence(a Evidence, b Evidence) bool {
	if a.FilePath != b.FilePath {
		return a.FilePath < b.FilePath
	}
	if a.Line != b.Line {
		return a.Line < b.Line
	}
	if a.Column != b.Column {
		return a.Column < b.Column
	}
	return a.Pattern < b.Pattern
}

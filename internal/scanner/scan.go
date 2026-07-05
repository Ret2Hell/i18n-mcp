package scanner

import (
	"cmp"
	"context"
	"maps"
	"os"
	"slices"
	"time"
)

// Scan scans source files for translation key usage.
func (s *Service) Scan(ctx context.Context, in ScanInput) (Report, error) {
	progress := in.Progress
	if progress == nil {
		progress = noopProgress{}
	}
	files, err := s.DiscoverSourceFiles(ctx, in.Files)
	if err != nil {
		return Report{}, err
	}
	total := len(files)
	progress.Step(ctx, "discovered source files", 0, total)

	batchSize := in.BatchSize
	if batchSize <= 0 {
		batchSize = 25
	}
	report := Report{FilesScanned: total, Files: files, GeneratedAt: time.Now().UTC()}
	usageByID := map[string]*Usage{}
	for i, file := range files {
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
		for _, ev := range scanNamespaceUsages(file.Path, data) {
			addEvidence(usageByID, ev, "namespace_bound_call")
		}
		report.DynamicHints = append(report.DynamicHints, scanDynamicHints(file.Path, data)...)
		if (i+1)%batchSize == 0 || i+1 == total {
			progress.Step(ctx, "scanned source files", i+1, total)
		}
	}
	report.Usages = sortedUsages(usageByID)
	slices.SortFunc(report.DynamicHints, compareDynamicHint)
	s.storeLatest(report)
	return report, nil
}

type noopProgress struct{}

func (noopProgress) Step(context.Context, string, int, int) {}

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
	ids := slices.Sorted(maps.Keys(usageByID))
	out := make([]Usage, 0, len(ids))
	for _, id := range ids {
		usage := *usageByID[id]
		slices.SortFunc(usage.Evidence, compareEvidence)
		out = append(out, usage)
	}
	return out
}

func compareEvidence(a Evidence, b Evidence) int {
	return cmp.Or(
		cmp.Compare(a.FilePath, b.FilePath),
		cmp.Compare(a.Line, b.Line),
		cmp.Compare(a.Column, b.Column),
		cmp.Compare(a.Pattern, b.Pattern),
	)
}

func compareDynamicHint(a DynamicHint, b DynamicHint) int {
	return cmp.Or(
		cmp.Compare(a.FilePath, b.FilePath),
		cmp.Compare(a.Line, b.Line),
		cmp.Compare(a.Column, b.Column),
		cmp.Compare(a.Pattern, b.Pattern),
	)
}

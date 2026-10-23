package report

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/diff"
)

// WriteFullJSON streams all diff rows to a local writer without retaining them
// or caching the export. A failed export may leave partial output; callers must
// check the error before consuming the file. Returned metadata supports CI policy.
func (s *Service) WriteFullJSON(ctx context.Context, w io.Writer) (Report, error) {
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	cfg, err := s.config.Resolve(ctx)
	if err != nil {
		return Report{}, err
	}
	inv, err := s.locales.Inventory(ctx)
	if err != nil {
		return Report{}, err
	}
	metadata := inv
	metadata.Units = nil
	r := Report{GeneratedAt: time.Now().UTC(), ProjectRoot: s.projectRoot, Config: cfg, Inventory: metadata}
	encoder := json.NewEncoder(w)
	field := func(name string, value any) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := io.WriteString(w, ",\""+name+"\":"); err != nil {
			return err
		}
		return encoder.Encode(value)
	}
	if _, err := io.WriteString(w, "{\"generatedAt\":"); err != nil {
		return Report{}, err
	}
	if err := encoder.Encode(r.GeneratedAt); err != nil {
		return Report{}, err
	}
	for _, entry := range []struct {
		name  string
		value any
	}{{"projectRoot", r.ProjectRoot}, {"config", cfg}, {"inventory", metadata}} {
		if err := field(entry.name, entry.value); err != nil {
			return Report{}, err
		}
	}
	if _, err := io.WriteString(w, ",\"diff\":{\"items\":["); err != nil {
		return Report{}, err
	}
	first := true
	analyzed, err := s.diff.WalkInventory(ctx, inv, diff.Filter{}, func(item diff.KeyDiff) error {
		if !first {
			if _, err := io.WriteString(w, ","); err != nil {
				return err
			}
		}
		first = false
		return encoder.Encode(item)
	})
	if err != nil {
		return Report{}, err
	}
	if _, err := io.WriteString(w, "]"); err != nil {
		return Report{}, err
	}
	for _, entry := range []struct {
		name  string
		value any
	}{{"sourceLocale", analyzed.SourceLocale}, {"targetLocales", analyzed.TargetLocales}, {"summary", analyzed.Summary}, {"warnings", analyzed.Warnings}} {
		if err := field(entry.name, entry.value); err != nil {
			return Report{}, err
		}
	}
	if _, err := io.WriteString(w, "}"); err != nil {
		return Report{}, err
	}
	r.Diff = analyzed
	r.Warnings = collectWarnings(inv, analyzed)
	r.Summary = summarize(r)
	if err := field("summary", r.Summary); err != nil {
		return Report{}, err
	}
	if err := field("warnings", r.Warnings); err != nil {
		return Report{}, err
	}
	_, err = io.WriteString(w, "}\n")
	return r, err
}

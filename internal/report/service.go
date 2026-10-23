package report

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/diff"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/Ret2Hell/i18n-mcp/internal/resources"
	"github.com/Ret2Hell/i18n-mcp/internal/security"
)

// Notifier publishes resource update notifications.
type Notifier interface {
	Updated(ctx context.Context, uris ...string)
}

// Service generates audit reports.
type Service struct {
	projectRoot string
	config      *config.Service
	locales     *locale.Service
	diff        *diff.Service
	Notifier    Notifier

	latest security.Store[GenerateOutput]
}

// NewService creates a report generation service.
func NewService(projectRoot string, configService *config.Service, localeService *locale.Service, diffService *diff.Service) *Service {
	return &Service{projectRoot: projectRoot, config: configService, locales: localeService, diff: diffService}
}

// Generate builds and renders an audit report.
func (s *Service) Generate(ctx context.Context, in GenerateInput) (GenerateOutput, error) {
	format := cmp.Or(in.Format, FormatJSON)
	progress := in.Progress
	if progress == nil {
		progress = noopProgress{}
	}
	const totalProgressSteps = 3

	cfg, err := s.config.Resolve(ctx)
	if err != nil {
		return GenerateOutput{}, err
	}
	progress.Step(ctx, "loading locale inventory", 1, totalProgressSteps)
	inv, err := s.locales.Inventory(ctx)
	if err != nil {
		return GenerateOutput{}, err
	}
	progress.Step(ctx, "computing translation diff", 2, totalProgressSteps)
	var details []diff.KeyDiff
	var visit func(diff.KeyDiff) error
	if in.Full {
		visit = func(item diff.KeyDiff) error { details = append(details, item); return nil }
	}
	diffReport, err := s.diff.WalkInventory(ctx, inv, diff.Filter{}, visit)
	diffReport.Items = details
	if err != nil {
		return GenerateOutput{}, err
	}
	// Source scanning and dead-key analysis are intentionally disabled. In
	// particular, never reuse a partial usage scan for audit or prune decisions.

	report := Report{
		GeneratedAt: time.Now().UTC(),
		ProjectRoot: s.projectRoot,
		Config:      cfg,
		Inventory:   inv,
		Diff:        diffReport,
		Warnings:    collectWarnings(inv, diffReport),
	}
	report.Summary = summarize(report)
	if !in.Full {
		report = compactReport(report)
	}
	out := GenerateOutput{Report: report, Format: format}
	if err := ctx.Err(); err != nil {
		return GenerateOutput{}, err
	}
	progress.Step(ctx, "rendering report", 3, totalProgressSteps)
	switch format {
	case FormatJSON:
		out.Text, err = RenderJSON(report)
	case FormatMarkdown:
		out.Text, err = RenderMarkdown(report)
	default:
		return GenerateOutput{}, fmt.Errorf("unsupported report format %q", format)
	}
	if err != nil {
		return GenerateOutput{}, err
	}
	if err := ctx.Err(); err != nil {
		return GenerateOutput{}, err
	}
	latest := out
	latest.Report = compactReport(report)
	latest.Text = ""
	s.storeLatest(ctx, latest)
	if s.Notifier != nil {
		s.Notifier.Updated(ctx, resources.LatestReportURI)
	}
	return out, nil
}

type noopProgress struct{}

func (noopProgress) Step(context.Context, string, int, int) {}

// Latest returns the current subject's most recent generated report, if any.
func (s *Service) Latest(ctx context.Context) (GenerateOutput, bool, error) {
	out, ok, err := s.latest.Get(ctx, "latest")
	return cloneOutput(out), ok, err
}

func (s *Service) storeLatest(ctx context.Context, out GenerateOutput) {
	s.latest.Put(ctx, "latest", cloneOutput(out))
}

func collectWarnings(inv locale.Inventory, diffReport diff.Report) []string {
	warnings := []string{"Source usage scanning and dead-key analysis are temporarily disabled; failOnDeadKeys is not evaluated."}
	for _, warning := range inv.Warnings {
		warnings = append(warnings, warning.Code+": "+warning.Message)
	}
	warnings = append(warnings, diffReport.Warnings...)
	return uniqueSorted(warnings)
}

func summarize(r Report) Summary {
	summary := Summary{
		Locales:    len(r.Inventory.Locales),
		Namespaces: len(r.Inventory.Namespaces),
		Missing:    r.Diff.Summary.Missing,
		Stale:      r.Diff.Summary.Stale,
		Invalid:    r.Diff.Summary.Invalid,
		Extra:      r.Diff.Summary.Extra,
		Unknown:    r.Diff.Summary.Unknown,
		Warnings:   len(r.Warnings),
	}
	for code, count := range r.Inventory.CountsByLocale {
		if code == r.Inventory.SourceLocale {
			summary.SourceKeys += count
		} else {
			summary.TargetKeys += count
		}
	}
	return summary
}

func uniqueSorted(values []string) []string {
	slices.Sort(values)
	values = slices.DeleteFunc(values, func(value string) bool {
		return value == ""
	})
	return slices.Compact(values)
}

// compactReport removes full datasets before storing or returning network reports.
func compactReport(r Report) Report {
	r.DetailsOmitted = true
	r.DetailResource = resources.DiffURI
	r.Inventory.Units = nil
	r.Diff.Items = nil
	r.WarningsTotal = max(r.WarningsTotal, len(r.Warnings))
	if len(r.Warnings) > 20 {
		r.Warnings = r.Warnings[:20]
	}
	r.Diff.WarningsTotal = max(r.Diff.WarningsTotal, len(r.Diff.Warnings))
	if len(r.Diff.Warnings) > 20 {
		r.Diff.Warnings = r.Diff.Warnings[:20]
	}
	r.Inventory.WarningsTotal = max(r.Inventory.WarningsTotal, len(r.Inventory.Warnings))
	if len(r.Inventory.Warnings) > 20 {
		r.Inventory.Warnings = r.Inventory.Warnings[:20]
	}
	return r
}

func cloneOutput(out GenerateOutput) GenerateOutput {
	out.Report.Warnings = slices.Clone(out.Report.Warnings)
	return out
}

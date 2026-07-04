package report

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/deadkey"
	"github.com/Ret2Hell/i18n-mcp/internal/diff"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/Ret2Hell/i18n-mcp/internal/scanner"
)

type Service struct {
	projectRoot string
	config      *config.Service
	locales     *locale.Service
	diff        *diff.Service
	scanner     *scanner.Service
	deadKeys    *deadkey.Service

	latestMu sync.RWMutex
	latest   *GenerateOutput
}

func NewService(projectRoot string, configService *config.Service, localeService *locale.Service, diffService *diff.Service, scannerService *scanner.Service, deadKeyService *deadkey.Service) *Service {
	return &Service{projectRoot: projectRoot, config: configService, locales: localeService, diff: diffService, scanner: scannerService, deadKeys: deadKeyService}
}

func (s *Service) Generate(ctx context.Context, in GenerateInput) (GenerateOutput, error) {
	format := cmp.Or(in.Format, FormatJSON)
	progress := in.Progress
	if progress == nil {
		progress = noopProgress{}
	}
	const totalProgressSteps = 5

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
	diffReport, err := s.diff.Analyze(ctx)
	if err != nil {
		return GenerateOutput{}, err
	}
	progress.Step(ctx, "scanning source usage", 3, totalProgressSteps)
	usageReport, err := s.usage(ctx, in.RefreshUsage)
	if err != nil {
		return GenerateOutput{}, err
	}
	progress.Step(ctx, "building dead-key report", 4, totalProgressSteps)
	deadReport, err := s.deadKeys.Report(ctx, deadkey.ReportInput{RefreshUsage: in.RefreshUsage})
	if err != nil {
		return GenerateOutput{}, err
	}

	report := Report{
		GeneratedAt: time.Now().UTC(),
		ProjectRoot: s.projectRoot,
		Config:      cfg,
		Inventory:   inv,
		Diff:        diffReport,
		Usage:       usageReport,
		DeadKeys:    deadReport,
		Warnings:    collectWarnings(inv, diffReport, usageReport, deadReport),
	}
	report.Summary = summarize(report)
	out := GenerateOutput{Report: report, Format: format}
	progress.Step(ctx, "rendering report", 5, totalProgressSteps)
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
	s.storeLatest(out)
	return out, nil
}

type noopProgress struct{}

func (noopProgress) Step(context.Context, string, int, int) {}

func (s *Service) Latest() (GenerateOutput, bool) {
	s.latestMu.RLock()
	defer s.latestMu.RUnlock()
	if s.latest == nil {
		return GenerateOutput{}, false
	}
	return cloneOutput(*s.latest), true
}

func (s *Service) storeLatest(out GenerateOutput) {
	s.latestMu.Lock()
	defer s.latestMu.Unlock()
	s.latest = new(cloneOutput(out))
}

func (s *Service) usage(ctx context.Context, refresh bool) (scanner.Report, error) {
	if !refresh {
		if usage, ok := s.scanner.Latest(); ok {
			return usage, nil
		}
	}
	return s.scanner.Scan(ctx, scanner.ScanInput{})
}

func collectWarnings(inv locale.Inventory, diffReport diff.Report, usage scanner.Report, deadReport deadkey.Report) []string {
	var warnings []string
	for _, warning := range inv.Warnings {
		warnings = append(warnings, warning.Code+": "+warning.Message)
	}
	warnings = append(warnings, diffReport.Warnings...)
	warnings = append(warnings, usage.Warnings...)
	warnings = append(warnings, deadReport.Warnings...)
	return uniqueSorted(warnings)
}

func summarize(r Report) Summary {
	summary := Summary{
		Locales:        len(r.Inventory.Locales),
		Namespaces:     len(r.Inventory.Namespaces),
		Missing:        r.Diff.Summary.Missing,
		Stale:          r.Diff.Summary.Stale,
		Invalid:        r.Diff.Summary.Invalid,
		Extra:          r.Diff.Summary.Extra,
		Unknown:        r.Diff.Summary.Unknown,
		ProbablyUnused: r.DeadKeys.Summary.ProbablyUnused,
		MaybeDynamic:   r.DeadKeys.Summary.MaybeDynamic,
		Warnings:       len(r.Warnings),
	}
	for _, unit := range r.Inventory.Units {
		if unit.Locale == r.Inventory.SourceLocale {
			summary.SourceKeys++
		} else {
			summary.TargetKeys++
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

func cloneOutput(out GenerateOutput) GenerateOutput {
	out.Report.Warnings = slices.Clone(out.Report.Warnings)
	return out
}

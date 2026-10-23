package cli

import (
	"errors"
	"fmt"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/report"
	"github.com/spf13/cobra"
)

// AuditError indicates that the audit report violated the configured CI policy.
type AuditError struct {
	Failures []report.Failure
}

func (e *AuditError) Error() string {
	return fmt.Sprintf("i18n audit failed with %d failure condition(s)", len(e.Failures))
}

func newAuditCommand(opts *RootOptions) *cobra.Command {
	var refreshUsage bool
	var full bool
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Run non-interactive i18n audit for CI",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAudit(cmd, opts, refreshUsage, full)
		},
	}
	cmd.Flags().BoolVar(&full, "full", false, "include full detail rows in local report output; may be large")
	cmd.Flags().BoolVar(&refreshUsage, "refresh-usage", false, "deprecated: source usage scanning is temporarily disabled")
	_ = cmd.Flags().MarkDeprecated("refresh-usage", "source usage scanning and dead-key analysis are temporarily disabled; this flag has no effect")
	return cmd
}

func runAudit(cmd *cobra.Command, opts *RootOptions, refreshUsage, full bool) error {
	ctx := cmd.Context()
	application, err := app.New(ctx, app.Options{ProjectRoot: opts.Project, ConfigPath: opts.Config, LogLevel: opts.LogLevel})
	if err != nil {
		return err
	}

	format := report.Format(opts.Output)
	if format == "" || format == "table" {
		format = report.FormatMarkdown
	}

	out, err := application.Reports.Generate(ctx, report.GenerateInput{Format: format, RefreshUsage: refreshUsage, Full: full})
	if err != nil {
		return err
	}
	if _, err := fmt.Fprint(cmd.OutOrStdout(), out.Text); err != nil {
		return err
	}

	failures := report.EvaluatePolicy(out.Report, out.Report.Config.CI)
	if len(failures) > 0 {
		for _, failure := range failures {
			if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "audit failure: %s (%d)\n", failure.Message, failure.Count); err != nil {
				return err
			}
		}
		return &AuditError{Failures: failures}
	}
	return nil
}

// IsAuditError reports whether err is an AuditError.
func IsAuditError(err error) bool {
	auditErr, ok := errors.AsType[*AuditError](err)
	return ok && auditErr != nil
}

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
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Run non-interactive i18n audit for CI",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAudit(cmd, opts, refreshUsage)
		},
	}
	cmd.Flags().BoolVar(&refreshUsage, "refresh-usage", true, "rerun usage scan before generating the report")
	return cmd
}

func runAudit(cmd *cobra.Command, opts *RootOptions, refreshUsage bool) error {
	ctx := cmd.Context()
	application, err := app.New(ctx, app.Options{ProjectRoot: opts.Project, ConfigPath: opts.Config, LogLevel: opts.LogLevel})
	if err != nil {
		return err
	}

	format := report.Format(opts.Output)
	if format == "" || format == "table" {
		format = report.FormatMarkdown
	}

	out, err := application.Reports.Generate(ctx, report.GenerateInput{Format: format, RefreshUsage: refreshUsage})
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

func IsAuditError(err error) bool {
	auditErr, ok := errors.AsType[*AuditError](err)
	return ok && auditErr != nil
}

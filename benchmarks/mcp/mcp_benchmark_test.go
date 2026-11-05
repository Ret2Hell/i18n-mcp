//go:build mcpbench

package mcpbench

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"

	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/httpserver"
	"github.com/Ret2Hell/i18n-mcp/internal/mcpserver"
	"github.com/Ret2Hell/i18n-mcp/internal/pagination"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// These benchmarks use the production HTTP handler, SDK client, tool handlers,
// file loading, analysis, pagination, response budgets, and request-state signing.
// Only fixture creation and correctness checks are excluded from measurements.
type benchmarkProvider struct{ server *mcp.Server }

func (p benchmarkProvider) ServerForRequest(*http.Request) *mcp.Server { return p.server }

type mcpWorkload struct {
	name, tool, field, items string
	arguments                map[string]any
	limit                    int
}

func mcpWorkloads() []mcpWorkload {
	workloads := []mcpWorkload{{name: "health", tool: "i18n.health"}}
	for _, limit := range []int{50, 200} {
		workloads = append(workloads,
			mcpWorkload{
				name: fmt.Sprintf("locales/page_%d", limit), tool: "i18n.locales.list", field: "inventory", items: "units", limit: limit,
				arguments: map[string]any{"limit": limit, "includeUnits": true, "locales": []string{"fr"}},
			},
			mcpWorkload{
				name: fmt.Sprintf("diff/page_%d", limit), tool: "i18n.keys.diff", field: "report", items: "items", limit: limit,
				arguments: map[string]any{"limit": limit, "locales": []string{"fr"}},
			},
			mcpWorkload{
				name: fmt.Sprintf("plan/page_%d", limit), tool: "i18n.translation.plan", field: "batch", items: "items", limit: limit,
				arguments: map[string]any{"maxItems": limit, "locales": []string{"fr"}},
			},
		)
	}
	return workloads
}

func makeMCPBenchmarkFixture(tb testing.TB) string {
	tb.Helper()
	root := tb.TempDir()
	write := func(path string, value any) {
		tb.Helper()
		data, err := json.Marshal(value)
		if err != nil {
			tb.Fatal(err)
		}
		path = filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			tb.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			tb.Fatal(err)
		}
	}
	write(".i18n-mcp.json", map[string]any{
		"sourceLocale": "en", "targetLocales": []string{"fr"},
		"localeFiles": []string{"messages/{locale}/{namespace}.json"},
	})
	write("package.json", map[string]any{"name": "mcp-benchmark", "dependencies": map[string]string{"next": "16.0.0", "next-intl": "4.0.0"}})
	// 1,000 source keys in two namespaces; half already translated. All
	// messages contain a real interpolation placeholder and some non-ASCII text.
	for _, namespace := range []string{"common", "checkout"} {
		source, target := map[string]string{}, map[string]string{}
		for i := range 500 {
			key := fmt.Sprintf("label_%04d", i)
			source[key] = fmt.Sprintf("Hello {name}, item %d — café", i)
			if i%2 == 0 {
				target[key] = fmt.Sprintf("Bonjour {name}, article %d — café", i)
			}
		}
		write("messages/en/"+namespace+".json", source)
		write("messages/fr/"+namespace+".json", target)
	}
	return root
}

func connectMCPBenchmark(ctx context.Context, root string) (*mcp.ClientSession, func(), error) {
	application, err := app.New(ctx, app.Options{
		ProjectRoot: root, LogLevel: "error", RequestStateSecret: "benchmark-only-secret-not-for-use-in-production",
	})
	if err != nil {
		return nil, nil, err
	}
	handler, err := httpserver.NewHandler(httpserver.Config{}, benchmarkProvider{mcpserver.New(application)}, application.Logger)
	if err != nil {
		return nil, nil, err
	}
	httpServer := httptest.NewServer(handler)
	transport := http.DefaultTransport.(*http.Transport).Clone()
	httpClient := &http.Client{Transport: transport, Timeout: 30 * time.Second}
	client := mcp.NewClient(&mcp.Implementation{Name: "benchmark-client", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: httpServer.URL + "/mcp", HTTPClient: httpClient, DisableStandaloneSSE: true,
	}, nil)
	closeAll := func() {
		if session != nil {
			_ = session.Close()
		}
		transport.CloseIdleConnections()
		httpServer.Close()
	}
	if err != nil {
		closeAll()
		return nil, nil, err
	}
	return session, closeAll, nil
}

var errBenchmarkBusy = errors.New("server busy")

func callMCPBenchmark(ctx context.Context, session *mcp.ClientSession, workload mcpWorkload) (*mcp.CallToolResult, error) {
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: workload.tool, Arguments: workload.arguments})
	if err != nil {
		return nil, err
	}
	if result.IsError {
		for _, content := range result.Content {
			if text, ok := content.(*mcp.TextContent); ok && strings.HasPrefix(text.Text, "Server is busy with another project operation;") {
				return nil, errBenchmarkBusy
			}
		}
		data, _ := json.Marshal(result.Content)
		return nil, fmt.Errorf("%s returned tool error: %s", workload.tool, data)
	}
	return result, nil
}

func checkMCPBenchmark(tb testing.TB, session *mcp.ClientSession, workload mcpWorkload) {
	tb.Helper()
	result, err := callMCPBenchmark(tb.Context(), session, workload)
	if err != nil {
		tb.Fatal(err)
	}
	if workload.limit == 0 {
		return
	}
	data, err := json.Marshal(result)
	if err != nil || len(data) > pagination.ResponseBudget {
		tb.Fatalf("response exceeds budget or cannot encode: bytes=%d err=%v", len(data), err)
	}
	data, err = json.Marshal(result.StructuredContent)
	if err != nil {
		tb.Fatal(err)
	}
	var output map[string]struct {
		Items []jsontext.Value    `json:"items"`
		Units []jsontext.Value    `json:"units"`
		Page  pagination.Metadata `json:"page"`
	}
	if err := json.Unmarshal(data, &output); err != nil {
		tb.Fatal(err)
	}
	page := output[workload.field]
	items := page.Items
	if workload.items == "units" {
		items = page.Units
	}
	expected := workload.limit
	if workload.tool == "i18n.translation.plan" && workload.limit == 200 {
		// Real production ItemBudget truncates this fixture before the count limit.
		expected = 195
	}
	if len(items) != expected || page.Page.Returned != expected || !page.Page.Truncated || page.Page.NextCursor == "" {
		tb.Fatalf("%s: got %d items, want %d with continuation; page=%+v", workload.name, len(items), expected, page.Page)
	}
}

func TestMCPBenchmarkWorkloads(t *testing.T) {
	root := makeMCPBenchmarkFixture(t)
	session, closeAll, err := connectMCPBenchmark(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	defer closeAll()
	for _, workload := range mcpWorkloads() {
		t.Run(workload.name, func(t *testing.T) { checkMCPBenchmark(t, session, workload) })
	}
}

func reportMCPLatency(b *testing.B, durations []time.Duration) {
	if len(durations) == 0 {
		b.Fatal("no successful requests")
	}
	slices.Sort(durations)
	b.ReportMetric(float64(durations[(len(durations)-1)/2].Nanoseconds())/1e6, "p50-ms")
	b.ReportMetric(float64(durations[(len(durations)-1)*95/100].Nanoseconds())/1e6, "p95-ms")
}

func BenchmarkMCPHTTP(b *testing.B) {
	root := makeMCPBenchmarkFixture(b)
	for _, workload := range mcpWorkloads() {
		for _, mode := range []string{"warm", "saturated", "fresh_lifecycle"} {
			b.Run(workload.name+"/"+mode, func(b *testing.B) {
				session, closeAll, err := connectMCPBenchmark(b.Context(), root)
				if err != nil {
					b.Fatal(err)
				}
				defer closeAll()
				checkMCPBenchmark(b, session, workload)
				durations := make([]time.Duration, b.N)
				b.ReportAllocs()
				b.ResetTimer()
				var busy atomic.Int64
				switch mode {
				case "saturated":
					var index atomic.Int64
					b.RunParallel(func(pb *testing.PB) {
						for pb.Next() {
							i := index.Add(1) - 1
							start := time.Now()
							_, err := callMCPBenchmark(b.Context(), session, workload)
							if errors.Is(err, errBenchmarkBusy) {
								busy.Add(1)
								continue
							}
							durations[i] = time.Since(start)
							if err != nil {
								b.Error(err)
								return
							}
						}
					})
				default:
					for i := range b.N {
						start := time.Now()
						if mode == "fresh_lifecycle" {
							fresh, closeFresh, err := connectMCPBenchmark(b.Context(), root)
							if err != nil {
								b.Fatal(err)
							}
							_, err = callMCPBenchmark(b.Context(), fresh, workload)
							closeFresh()
							if err != nil {
								b.Fatal(err)
							}
						} else if _, err := callMCPBenchmark(b.Context(), session, workload); err != nil {
							b.Fatal(err)
						}
						durations[i] = time.Since(start)
					}
				}
				b.StopTimer()
				b.ReportMetric(float64(busy.Load())/float64(b.N)*100, "busy-%")
				b.ReportMetric(float64(int64(b.N)-busy.Load())/b.Elapsed().Seconds(), "success/s")
				durations = slices.DeleteFunc(durations, func(d time.Duration) bool { return d == 0 })
				reportMCPLatency(b, durations)
			})
		}
	}
}

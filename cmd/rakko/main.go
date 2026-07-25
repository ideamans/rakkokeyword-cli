// Command rakko is a CLI for the ラッコキーワード (rakkokeyword) API.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ideamans/go-llm-cli-kit/llmcmd"
	"github.com/spf13/cobra"

	"github.com/ideamans/rakkokeyword-cli/internal/config"
	"github.com/ideamans/rakkokeyword-cli/internal/output"
	"github.com/ideamans/rakkokeyword-cli/internal/rakko"
)

//go:generate go run . gen-llmdocs

// PluginVersion is the released version of this CLI. It is also the version
// recorded in plugins/rakkokeyword-cli/.claude-plugin/plugin.json — a test
// enforces that the two agree, and the release workflow enforces that both
// agree with the git tag. Bump it in the same commit as the tag.
const PluginVersion = "0.1.0"

func main() {
	// --llm anywhere on the command line prints the reference and exits,
	// bypassing cobra so it keeps working regardless of subcommand position.
	if handled, err := llmcmd.HandleLegacy(os.Args[1:], llmConfig(), os.Stdout); handled {
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		return
	}

	rootCmd.AddCommand(newGenerateCommand())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:     "rakko",
	Short:   "Keyword research with the rakkokeyword (ラッコキーワード) API",
	Version: PluginVersion,
	Long: "Keyword research, SEO metrics, competitor and SERP data from the\n" +
		"rakkokeyword (ラッコキーワード) API.\n\n" +
		"Calls consume credits from the account the API key belongs to. Every\n" +
		"command states its cost, and --dry-run prints the request without\n" +
		"spending anything.\n\n" +
		"The full agent-facing reference is embedded in the binary: `rakko llm`.",

	// A rejected flag value or a failed API call is not a reason to reprint
	// the whole usage block; the error line is the useful part.
	SilenceUsage: true,
}

// Global flags. Every command shares them, so an agent that learns them once
// can drive the whole CLI.
var (
	flagAPIKey  string
	flagFormat  string
	flagFields  string
	flagDryRun  bool
	flagTimeout time.Duration
	flagRetries int
	flagWide    bool
	flagQuiet   bool
)

func init() {
	f := rootCmd.PersistentFlags()
	f.StringVar(&flagAPIKey, "api-key", "", "API key (overrides RAKKOKEYWORD_API_KEY and the config file)")
	f.StringVarP(&flagFormat, "format", "f", "", "Output format: table / json / jsonl / csv (default table, or the config file default)")
	f.StringVar(&flagFields, "fields", "", "Comma-separated columns for table and csv, as dotted paths into a record (e.g. keyword,metrics.searchVolume)")
	f.BoolVar(&flagDryRun, "dry-run", false, "Print the request that would be sent, consume no credits, and exit")
	f.DurationVar(&flagTimeout, "timeout", 120*time.Second, "HTTP timeout per request")
	f.IntVar(&flagRetries, "retries", 3, "Extra attempts for rate limits (429) and server errors (5xx)")
	f.BoolVar(&flagWide, "wide", false, "Do not truncate long values in table output")
	f.BoolVar(&flagQuiet, "quiet", false, "Suppress the credit-consumption notice on stderr")
}

// call is one API invocation plus how to render and price it.
type call struct {
	req rakko.Request
	out output.Options

	// credits is the documented cost, printed by --dry-run so the user can see
	// what a command would spend before spending it.
	credits string
}

// resolveFormat applies --format, then the config default, then table.
func resolveFormat(cfg *config.Config) (string, error) {
	format := flagFormat
	if format == "" {
		format = cfg.DefaultFormat
	}
	if format == "" {
		format = "table"
	}
	return resolveFormatValue(format)
}

// resolveFormatValue validates one format name.
func resolveFormatValue(format string) (string, error) {
	if !output.ValidFormat(format) {
		return "", fmt.Errorf("invalid format %q: must be one of %s", format, strings.Join(output.Formats, ", "))
	}
	return format, nil
}

// run performs one call and renders the result.
func run(cmd *cobra.Command, c call) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	format, err := resolveFormat(cfg)
	if err != nil {
		return err
	}
	c.out.Format = format
	c.out.Wide = flagWide
	if flagFields != "" {
		c.out.Fields = rakko.SplitList([]string{flagFields})
	}

	client := rakko.New(cfg.APIKeyResolved(flagAPIKey), cfg.BaseURLResolved(), flagTimeout, flagRetries)

	if flagDryRun {
		return printDryRun(cmd, client, c)
	}

	resp, err := client.Do(ctx(cmd), c.req)
	if err != nil {
		return err
	}
	report(cmd, resp)
	return output.Render(cmd.OutOrStdout(), resp.Raw, c.out)
}

// printDryRun shows the request without sending it. It is printed as JSON so
// an agent can inspect exactly what a command would have posted.
func printDryRun(cmd *cobra.Command, client *rakko.Client, c call) error {
	preview := map[string]any{
		"method": c.req.Method,
		"url":    client.URL(c.req),
		"cost":   c.credits,
	}
	if c.req.Body != nil {
		preview["body"] = c.req.Body
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(preview)
}

// report writes the credit cost and any non-fatal errors to stderr, keeping
// stdout parseable.
func report(cmd *cobra.Command, resp *rakko.Response) {
	for _, e := range resp.Errors {
		fmt.Fprintln(cmd.ErrOrStderr(), "API error:", e)
	}
	if flagQuiet {
		return
	}
	if resp.Meta.ConsumedCredit > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "consumed %g credit(s)\n", resp.Meta.ConsumedCredit)
	}
}

// ── shared list flags ────────────────────────────────────────────────────────

// listFlags are the sort / limit / filter flags most search endpoints share.
type listFlags struct {
	sortBy     string
	orderBy    string
	limit      int
	filters    []string
	filterJSON string

	sortValues []string
	spec       rakko.FilterSpec
}

// addTo registers the flags this endpoint actually supports. Pass a nil spec
// for endpoints without a filter object, an empty sortValues for endpoints
// without sorting, and an empty limitNote for endpoints without a limit.
func (l *listFlags) addTo(cmd *cobra.Command, sortValues []string, sortDefault, orderDefault, limitNote string, spec rakko.FilterSpec) {
	l.sortValues = sortValues
	l.spec = spec
	f := cmd.Flags()

	if len(sortValues) > 0 {
		f.StringVar(&l.sortBy, "sort-by", "", fmt.Sprintf("Sort field: %s (API default: %s)", strings.Join(sortValues, " / "), sortDefault))
		f.StringVar(&l.orderBy, "order-by", "", fmt.Sprintf("Sort order: asc / desc (API default: %s)", orderDefault))
	}
	if limitNote != "" {
		f.IntVarP(&l.limit, "limit", "n", 0, "Maximum records to return, "+limitNote)
	}
	if len(spec) > 0 {
		f.StringArrayVar(&l.filters, "filter", nil, "Filter as path=value, repeatable. Accepted paths:\n"+spec.Usage())
		f.StringVar(&l.filterJSON, "filter-json", "", "Filter object as raw JSON, merged over --filter (escape hatch for anything --filter cannot express)")
	}
}

// applyTo validates the flags and writes them into the request body.
func (l *listFlags) applyTo(cmd *cobra.Command, body rakko.Body) error {
	if err := rakko.Enum("sort-by", l.sortBy, l.sortValues); err != nil {
		return err
	}
	if err := rakko.Enum("order-by", l.orderBy, []string{"asc", "desc"}); err != nil {
		return err
	}
	body.SetIf(cmd.Flags().Changed("sort-by"), "sortBy", l.sortBy)
	body.SetIf(cmd.Flags().Changed("order-by"), "orderBy", l.orderBy)
	body.SetIf(cmd.Flags().Changed("limit"), "limit", l.limit)

	filters, err := l.spec.ParseFilters(l.filters)
	if err != nil {
		return err
	}
	filters, err = rakko.MergeFilterJSON(filters, l.filterJSON)
	if err != nil {
		return err
	}
	if len(filters) > 0 {
		body.Set("filter", filters)
	}
	return nil
}

// ── shared helpers ───────────────────────────────────────────────────────────

// allowedInt validates a flag against the fixed set of values the API accepts.
func allowedInt(flag string, value int, allowed []int) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	parts := make([]string, len(allowed))
	for i, a := range allowed {
		parts[i] = fmt.Sprint(a)
	}
	return fmt.Errorf("invalid --%s %d: must be one of %s", flag, value, strings.Join(parts, ", "))
}

// readLines reads one value per line from a file, or from stdin when the path
// is "-". Blank lines and # comments are skipped, so a keyword list can carry
// notes.
func readLines(path string) ([]string, error) {
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out, nil
}

// collectValues merges repeated flag values with an optional file, keeping the
// order given and dropping duplicates.
func collectValues(direct []string, file string) ([]string, error) {
	values := append([]string(nil), direct...)
	if file != "" {
		fromFile, err := readLines(file)
		if err != nil {
			return nil, err
		}
		values = append(values, fromFile...)
	}
	seen := map[string]bool{}
	var out []string
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out, nil
}

// ctx returns the command's context, or a background one in tests.
func ctx(cmd *cobra.Command) context.Context {
	if c := cmd.Context(); c != nil {
		return c
	}
	return context.Background()
}

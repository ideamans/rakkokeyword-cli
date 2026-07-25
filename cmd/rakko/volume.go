package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ideamans/rakkokeyword-cli/internal/output"
	"github.com/ideamans/rakkokeyword-cli/internal/rakko"
)

var searchVolumeCmd = &cobra.Command{
	Use:     "search-volume",
	Aliases: []string{"volume"},
	Short:   "Batch keyword metrics: fresh search volume, SEO difficulty, CPC, trends",
	Long: "Batch investigation of a keyword list — monthly search volume, SEO\n" +
		"difficulty, CPC, competition and month-by-month trends — as an asynchronous\n" +
		"job: register, poll status, fetch results.\n\n" +
		"This is the only command that returns freshly measured metrics. Everywhere\n" +
		"else the metrics ride along with whatever was last cached.\n\n" +
		"`rakko search-volume register --wait` runs all three steps in one go.",
}

func init() { rootCmd.AddCommand(searchVolumeCmd) }

// ── rakko search-volume register ─────────────────────────────────────────────

var (
	volumeKeywords    []string
	volumeFile        string
	volumeSEO         bool
	volumeCompletion  bool
	volumeLocation    string
	volumeLanguage    string
	volumeDedupe      bool
	volumeAggregation int
	volumeWait        waitFlags
	volumeResultsList listFlags
	volumeNoiseRed    bool
)

var volumeRegisterCmd = &cobra.Command{
	Use:   "register [keyword...]",
	Short: "Register a batch keyword investigation (0.03 credits/keyword, 0.78 with --seo-difficulty, minimum 15)",
	Long: "Registers up to 50,000 keywords for investigation and prints the requestId.\n\n" +
		"Without --wait, follow up with `rakko search-volume status <requestId>` and\n" +
		"then `rakko search-volume results <requestId>`. With --wait this command\n" +
		"polls the status itself and prints the results when they are ready; the\n" +
		"result-shaping flags (--limit, --sort-by, --filter, --noise-reduction)\n" +
		"apply to that final fetch.\n\n" +
		"Timing: usually about 10 seconds, but --seo-difficulty pushes it to as much\n" +
		"as 60 minutes, and a busy queue can take hours. Leave --seo-difficulty off\n" +
		"unless the difficulty score is what you came for.\n\n" +
		"Cost: 0.03 credits per keyword, plus 0.75 per keyword with --seo-difficulty.\n" +
		"A request always costs at least 15 credits, so batching pays.",
	Example: "  rakko search-volume register ラッコ カワウソ --wait\n" +
		"  rakko search-volume register --keywords-file keywords.txt --seo-difficulty\n" +
		"  rakko volume register --keywords-file - --location Japan --language Japanese",
	RunE: func(cmd *cobra.Command, args []string) error {
		keywords, err := collectValues(append(args, volumeKeywords...), volumeFile)
		if err != nil {
			return err
		}
		if len(keywords) == 0 {
			return fmt.Errorf("at least one keyword is required (as an argument, --keyword or --keywords-file)")
		}
		if len(keywords) > 50000 {
			return fmt.Errorf("too many keywords: %d (the API accepts at most 50,000)", len(keywords))
		}
		if cmd.Flags().Changed("aggregation-period-months") {
			if err := allowedInt("aggregation-period-months", volumeAggregation, []int{12, 24, 36, 48}); err != nil {
				return err
			}
		}

		body := rakko.Body{"keywords": keywords}
		body.SetIf(cmd.Flags().Changed("seo-difficulty"), "seoDifficulty", volumeSEO)
		body.SetIf(cmd.Flags().Changed("data-completion"), "dataCompletion", volumeCompletion)
		body.SetIf(cmd.Flags().Changed("location"), "location", volumeLocation)
		body.SetIf(cmd.Flags().Changed("language"), "language", volumeLanguage)
		body.SetIf(cmd.Flags().Changed("deduplicate"), "deduplicate", volumeDedupe)
		body.SetIf(cmd.Flags().Changed("aggregation-period-months"), "aggregationPeriodMonths", volumeAggregation)

		cost := 0.03 * float64(len(keywords))
		if volumeSEO {
			cost += 0.75 * float64(len(keywords))
		}
		credits := fmt.Sprintf("%.2f credits for %d keyword(s), minimum 15", maxFloat(15, cost), len(keywords))

		registerCall := call{
			req:     rakko.Request{Method: "POST", Path: "/v1/search-volume", Body: body},
			credits: credits,
			out:     output.Options{Caption: []string{"data.requestId"}},
		}
		if !volumeWait.wait || flagDryRun {
			return run(cmd, registerCall)
		}

		// --wait: register, poll, then fetch the results in one invocation.
		client, err := clientFor()
		if err != nil {
			return err
		}
		resp, err := client.Do(ctx(cmd), registerCall.req)
		if err != nil {
			return err
		}
		report(cmd, resp)
		requestID, err := requestIDOf(resp)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "requestId %s registered (%d keywords)\n", requestID, len(keywords))
		if err := pollUntilComplete(cmd, client, "/v1/search-volume/"+requestID+"/status", volumeWait); err != nil {
			return err
		}
		return runVolumeResults(cmd, requestID)
	},
}

func init() {
	searchVolumeCmd.AddCommand(volumeRegisterCmd)
	f := volumeRegisterCmd.Flags()
	f.StringArrayVar(&volumeKeywords, "keyword", nil, "Keyword to investigate; repeat, or pass keywords as arguments")
	f.StringVar(&volumeFile, "keywords-file", "", "File with one keyword per line (- for stdin); up to 50,000")
	f.BoolVar(&volumeSEO, "seo-difficulty", false, "Also measure SEO difficulty — 25x the cost and up to 60 minutes slower")
	f.BoolVar(&volumeCompletion, "data-completion", true, "Fill gaps in the volume data (API default: true)")
	f.StringVar(&volumeLocation, "location", "", "Region name, from `rakko metadata locations` (API default: Japan)")
	f.StringVar(&volumeLanguage, "language", "", "Language name, from `rakko metadata languages` (API default: Japanese)")
	f.BoolVar(&volumeDedupe, "deduplicate", true, "Drop duplicate keywords before charging for them (API default: true)")
	f.IntVar(&volumeAggregation, "aggregation-period-months", 0, "Trend window in months: 12 / 24 / 36 / 48 (API default: 12)")
	volumeWait.addTo(volumeRegisterCmd, "; the result flags below then shape the fetched results")
	addVolumeResultFlags(volumeRegisterCmd)
}

// ── rakko search-volume results ──────────────────────────────────────────────

// addVolumeResultFlags is shared by `results` and by `register --wait`.
func addVolumeResultFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&volumeNoiseRed, "noise-reduction", true, "Apply noise reduction to the result set (API default: true)")
	volumeResultsList.addTo(cmd,
		[]string{"keyword", "seoDifficulty", "searchVolume", "rateOfChange", "cpc", "competition", "firstSeenRange"},
		"searchVolume", "desc", "1-50000 (API default: 100)", rakko.SearchVolumeResultFilters)
}

var volumeResultsCmd = &cobra.Command{
	Use:   "results <requestId>",
	Short: "Fetch the results of a completed batch keyword investigation (free)",
	Long: "Fetches the data for a requestId from `rakko search-volume register`.\n" +
		"Check `rakko search-volume status <requestId>` first — results before\n" +
		"completion are partial.\n\n" +
		"Cost: free.",
	Example: "  rakko search-volume results 1234567 -n 500 -f csv\n" +
		"  rakko volume results 1234567 --filter searchVolume.min=100 --sort-by searchVolume",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runVolumeResults(cmd, args[0])
	},
}

func runVolumeResults(cmd *cobra.Command, requestID string) error {
	body := rakko.Body{}
	body.SetIf(cmd.Flags().Changed("noise-reduction"), "noiseReduction", volumeNoiseRed)
	if err := volumeResultsList.applyTo(cmd, body); err != nil {
		return err
	}
	return run(cmd, call{
		req:     rakko.Request{Method: "POST", Path: "/v1/search-volume/" + requestID + "/results", Body: body},
		credits: "free",
		out: output.Options{
			ItemsPath: "data.items",
			Caption: []string{"data.query.requestId", "data.query.location", "data.query.language",
				"data.summary.totalCount", "data.summary.returnedCount"},
			Columns: []string{
				"keyword", "metrics.searchVolume", "metrics.seoDifficulty", "metrics.cpc",
				"metrics.competition", "trends.changeRate.12m", "dataSource",
			},
		},
	})
}

func init() {
	searchVolumeCmd.AddCommand(volumeResultsCmd)
	addVolumeResultFlags(volumeResultsCmd)
}

// ── rakko search-volume status ───────────────────────────────────────────────

var volumeStatusWait waitFlags

var volumeStatusCmd = &cobra.Command{
	Use:   "status <requestId>",
	Short: "Check whether a batch keyword investigation has finished (free)",
	Long: "Reports isCompleted plus the per-stage statuses. Poll every 30 seconds;\n" +
		"--wait does that for you and returns when the job is done.\n\n" +
		"noiseReduction is not part of the completion test — isCompleted can be true\n" +
		"while it is still processing.\n\n" +
		"Cost: free.",
	Example: "  rakko search-volume status 1234567\n" +
		"  rakko search-volume status 1234567 --wait",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "/v1/search-volume/" + args[0] + "/status"
		if volumeStatusWait.wait && !flagDryRun {
			client, err := clientFor()
			if err != nil {
				return err
			}
			if err := pollUntilComplete(cmd, client, path, volumeStatusWait); err != nil {
				return err
			}
		}
		return run(cmd, call{
			req:     rakko.Request{Method: "GET", Path: path},
			credits: "free",
			out:     output.Options{DataPath: "data"},
		})
	},
}

func init() {
	searchVolumeCmd.AddCommand(volumeStatusCmd)
	volumeStatusWait.addTo(volumeStatusCmd, "")
}

// ── rakko search-volume histories ────────────────────────────────────────────

var (
	volumeHistLimit  int
	volumeHistOffset int
	volumeHistStatus string
)

var volumeHistoriesCmd = &cobra.Command{
	Use:   "histories",
	Short: "List past batch keyword investigations, newest first (free)",
	Long: "Past requests with their requestId, status and keyword summary, newest\n" +
		"first. Use it to recover a requestId whose results were never fetched.\n\n" +
		"Cost: free.",
	Example: "  rakko search-volume histories -n 20\n" +
		"  rakko volume histories --status processing",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		q, err := historiesQuery(cmd, volumeHistLimit, volumeHistOffset, volumeHistStatus)
		if err != nil {
			return err
		}
		return run(cmd, call{
			req:     rakko.Request{Method: "GET", Path: "/v1/search-volume/histories", Query: q},
			credits: "free",
			out: output.Options{
				ItemsPath: "data.items",
				Caption:   []string{"data.summary.totalCount", "data.summary.returnedCount"},
				Columns: []string{
					"requestId", "status", "createdAt", "completedAt",
					"keywordCount", "seoDifficulty", "keywordSummary",
				},
			},
		})
	},
}

func init() {
	searchVolumeCmd.AddCommand(volumeHistoriesCmd)
	f := volumeHistoriesCmd.Flags()
	f.IntVarP(&volumeHistLimit, "limit", "n", 0, "Maximum records to return, 1-100 (API default: 100)")
	f.IntVar(&volumeHistOffset, "offset", 0, "Records to skip (offset + limit must not exceed 50,000)")
	f.StringVar(&volumeHistStatus, "status", "", "Filter by status: completed / processing (API default: all)")
}

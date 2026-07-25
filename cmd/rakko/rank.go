package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ideamans/rakkokeyword-cli/internal/output"
	"github.com/ideamans/rakkokeyword-cli/internal/rakko"
)

var searchRankCmd = &cobra.Command{
	Use:     "search-rank",
	Aliases: []string{"rank"},
	Short:   "Check live Google rankings of URLs for a list of keywords",
	Long: "Measures where URLs or domains currently rank in Google for given keywords,\n" +
		"as an asynchronous job: register, poll status, fetch results.\n\n" +
		"Unlike the ranks bundled with `rakko influx-keywords`, these are freshly\n" +
		"fetched SERPs for the region, language, device and OS you specify.\n\n" +
		"`rakko search-rank register --wait` runs all three steps in one go.",
}

func init() { rootCmd.AddCommand(searchRankCmd) }

// ── rakko search-rank register ───────────────────────────────────────────────

var (
	rankKeywords    []string
	rankKeywordFile string
	rankURLs        []string
	rankURLFile     string
	rankMatchType   string
	rankDepth       int
	rankWithVolume  bool
	rankDedupe      bool
	rankLocation    string
	rankLanguage    string
	rankDevice      string
	rankOS          string
	rankWait        waitFlags

	rankResultsList listFlags
	rankAggregation bool
	rankStatusWait  waitFlags
	rankHistLimit   int
	rankHistOffset  int
	rankHistStatus  string
)

var rankRegisterCmd = &cobra.Command{
	Use:   "register [keyword...]",
	Short: "Register a rank check for keywords against URLs (0.9 credits/keyword for ranks 1-30)",
	Long: "Registers a rank check and prints the requestId. Every keyword is checked\n" +
		"against every URL, so 10 keywords and 3 URLs is one job covering 30 pairs —\n" +
		"but the cost is per keyword, not per pair.\n\n" +
		"Without --wait, follow up with `rakko search-rank status <requestId>` and\n" +
		"then `rakko search-rank results <requestId>`. With --wait this command\n" +
		"polls and prints the results itself.\n\n" +
		"Timing: up to 10 keywords usually finishes in minutes, larger jobs within\n" +
		"about an hour, and a busy queue can take longer.\n\n" +
		"Cost: 0.9 credits per keyword for ranks 1-30, plus 0.3 per keyword for each\n" +
		"additional 10 ranks of --depth. --search-volume adds the metrics of\n" +
		"`rakko search-volume` on top.",
	Example: "  rakko search-rank register ラッコ カワウソ --url https://example.com/ --wait\n" +
		"  rakko search-rank register --keywords-file kw.txt --url https://example.com/ --depth 100 --device mobile",
	RunE: func(cmd *cobra.Command, args []string) error {
		keywords, err := collectValues(append(args, rankKeywords...), rankKeywordFile)
		if err != nil {
			return err
		}
		urls, err := collectValues(rankURLs, rankURLFile)
		if err != nil {
			return err
		}
		if len(keywords) == 0 {
			return fmt.Errorf("at least one keyword is required (as an argument, --keyword or --keywords-file)")
		}
		if len(urls) == 0 {
			return fmt.Errorf("at least one --url (or --urls-file) is required")
		}
		if len(urls) > 50 {
			return fmt.Errorf("too many URLs: %d (the API accepts at most 50)", len(urls))
		}
		if err := rakko.Enum("match-type", rankMatchType, urlMatchTypes); err != nil {
			return err
		}
		if cmd.Flags().Changed("depth") {
			if err := allowedInt("depth", rankDepth, []int{30, 40, 50, 60, 70, 80, 90, 100}); err != nil {
				return err
			}
		}
		if err := rakko.Enum("device", rankDevice, []string{"desktop", "mobile"}); err != nil {
			return err
		}
		if err := rakko.Enum("os", rankOS, []string{"windows", "macos", "android", "ios"}); err != nil {
			return err
		}

		body := rakko.Body{"keywords": keywords, "urls": urls}
		body.SetIf(cmd.Flags().Changed("match-type"), "matchType", rankMatchType)
		body.SetIf(cmd.Flags().Changed("depth"), "depth", rankDepth)
		body.SetIf(cmd.Flags().Changed("search-volume"), "isSearchVolumeAndSeoDifficultyEnabled", rankWithVolume)
		body.SetIf(cmd.Flags().Changed("deduplicate"), "deduplicate", rankDedupe)
		body.SetIf(cmd.Flags().Changed("location"), "location", rankLocation)
		body.SetIf(cmd.Flags().Changed("language"), "language", rankLanguage)
		body.SetIf(cmd.Flags().Changed("device"), "device", rankDevice)
		body.SetIf(cmd.Flags().Changed("os"), "os", rankOS)

		depth := rankDepth
		if depth == 0 {
			depth = 30
		}
		perKeyword := 0.9 + 0.3*float64((depth-30)/10)
		credits := fmt.Sprintf("%.2f credits for %d keyword(s) to rank %d (%.2f per keyword)",
			perKeyword*float64(len(keywords)), len(keywords), depth, perKeyword)

		registerCall := call{
			req:     rakko.Request{Method: "POST", Path: "/v1/search-rank", Body: body},
			credits: credits,
			out:     output.Options{Caption: []string{"data.requestId"}},
		}
		if !rankWait.wait || flagDryRun {
			return run(cmd, registerCall)
		}

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
		fmt.Fprintf(cmd.ErrOrStderr(), "requestId %s registered (%d keywords × %d URLs)\n", requestID, len(keywords), len(urls))
		if err := pollUntilComplete(cmd, client, "/v1/search-rank/"+requestID+"/status", rankWait); err != nil {
			return err
		}
		return runRankResults(cmd, requestID)
	},
}

func init() {
	searchRankCmd.AddCommand(rankRegisterCmd)
	f := rankRegisterCmd.Flags()
	f.StringArrayVar(&rankKeywords, "keyword", nil, "Keyword to check; repeat, or pass keywords as arguments")
	f.StringVar(&rankKeywordFile, "keywords-file", "", "File with one keyword per line (- for stdin)")
	f.StringArrayVar(&rankURLs, "url", nil, "URL or domain to look for in the results; repeat, up to 50")
	f.StringVar(&rankURLFile, "urls-file", "", "File with one URL per line")
	f.StringVar(&rankMatchType, "match-type", "", "How a result counts as a hit: url / forward_url / domain / sub_domain (API default: sub_domain)")
	f.IntVar(&rankDepth, "depth", 0, "How deep to read the SERP: 30 / 40 / … / 100 (API default: 30)")
	f.BoolVar(&rankWithVolume, "search-volume", false, "Also fetch search volume and SEO difficulty for every keyword (extra credits)")
	f.BoolVar(&rankDedupe, "deduplicate", true, "Drop duplicate keywords before charging for them (API default: true)")
	f.StringVar(&rankLocation, "location", "", "Region name for the SERP, from `rakko metadata locations` (API default: Japan)")
	f.StringVar(&rankLanguage, "language", "", "Language name for the SERP, from `rakko metadata languages` (API default: Japanese)")
	f.StringVar(&rankDevice, "device", "", "Device to emulate: desktop / mobile (API default: desktop)")
	f.StringVar(&rankOS, "os", "", "OS to emulate: windows / macos (desktop) or android / ios (mobile)")
	rankWait.addTo(rankRegisterCmd, "; the result flags below then shape the fetched results")
	addRankResultFlags(rankRegisterCmd)
}

// ── rakko search-rank results ────────────────────────────────────────────────

func addRankResultFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&rankAggregation, "with-aggregation", false, "Include per-target totals (estimated traffic, rank distribution) in the summary")
	rankResultsList.addTo(cmd,
		[]string{"keyword", "seoDifficulty", "searchVolume"},
		"searchVolume", "desc", "any positive integer (API default: 100)", rakko.SearchRankResultFilters)
}

var rankResultsCmd = &cobra.Command{
	Use:   "results <requestId>",
	Short: "Fetch the results of a completed rank check (free)",
	Long: "Fetches the ranks for a requestId from `rakko search-rank register`.\n\n" +
		"Each item carries a rankings array with one entry per target URL. A null\n" +
		"position means the URL was not found within --depth — that is \"not in the\n" +
		"top N\", not \"rank 0\". In table output the rankings array is shown as JSON;\n" +
		"use -f json or --fields to work with it properly.\n\n" +
		"Cost: free.",
	Example: "  rakko search-rank results 01HQZX5Y4JMQK8XNQ7WVZXZ5Y4\n" +
		"  rakko search-rank results 01HQZX… --with-aggregation -f json",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRankResults(cmd, args[0])
	},
}

func runRankResults(cmd *cobra.Command, requestID string) error {
	body := rakko.Body{}
	body.SetIf(cmd.Flags().Changed("with-aggregation"), "withAggregation", rankAggregation)
	if err := rankResultsList.applyTo(cmd, body); err != nil {
		return err
	}
	return run(cmd, call{
		req:     rakko.Request{Method: "POST", Path: "/v1/search-rank/" + requestID + "/results", Body: body},
		credits: "free",
		out: output.Options{
			ItemsPath: "data.items",
			Caption:   []string{"data.query.requestId", "data.summary.totalCount", "data.summary.returnedCount"},
			Columns: []string{
				"keyword", "metrics.searchVolume", "metrics.seoDifficulty", "rankings",
			},
		},
	})
}

func init() {
	searchRankCmd.AddCommand(rankResultsCmd)
	addRankResultFlags(rankResultsCmd)
}

// ── rakko search-rank status ─────────────────────────────────────────────────

var rankStatusCmd = &cobra.Command{
	Use:   "status <requestId>",
	Short: "Check whether a rank check has finished (free)",
	Long: "Reports isCompleted plus the per-stage statuses. Poll every 30 seconds;\n" +
		"--wait does that for you.\n\n" +
		"A searchVolumeAndSeoDifficulty status of failed or integration_failed keeps\n" +
		"isCompleted false — the SERP ranks may still be fetchable.\n\n" +
		"Cost: free.",
	Example: "  rakko search-rank status 01HQZX5Y4JMQK8XNQ7WVZXZ5Y4 --wait",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "/v1/search-rank/" + args[0] + "/status"
		if rankStatusWait.wait && !flagDryRun {
			client, err := clientFor()
			if err != nil {
				return err
			}
			if err := pollUntilComplete(cmd, client, path, rankStatusWait); err != nil {
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
	searchRankCmd.AddCommand(rankStatusCmd)
	rankStatusWait.addTo(rankStatusCmd, "")
}

// ── rakko search-rank histories ──────────────────────────────────────────────

var rankHistoriesCmd = &cobra.Command{
	Use:   "histories",
	Short: "List past rank checks, newest first (free)",
	Long: "Past rank checks with their requestId, status, keyword and URL summaries,\n" +
		"newest first. Use it to recover a requestId.\n\n" +
		"Cost: free.",
	Example: "  rakko search-rank histories -n 20 --status completed",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		q, err := historiesQuery(cmd, rankHistLimit, rankHistOffset, rankHistStatus)
		if err != nil {
			return err
		}
		return run(cmd, call{
			req:     rakko.Request{Method: "GET", Path: "/v1/search-rank/histories", Query: q},
			credits: "free",
			out: output.Options{
				ItemsPath: "data.items",
				Caption:   []string{"data.summary.totalCount", "data.summary.returnedCount"},
				Columns: []string{
					"requestId", "status", "createdAt", "completedAt",
					"keywordCount", "urlCount", "matchType", "depth", "keywordSummary",
				},
			},
		})
	},
}

func init() {
	searchRankCmd.AddCommand(rankHistoriesCmd)
	f := rankHistoriesCmd.Flags()
	f.IntVarP(&rankHistLimit, "limit", "n", 0, "Maximum records to return, 1-100 (API default: 100)")
	f.IntVar(&rankHistOffset, "offset", 0, "Records to skip (offset + limit must not exceed 50,000)")
	f.StringVar(&rankHistStatus, "status", "", "Filter by status: completed / processing (API default: all)")
}

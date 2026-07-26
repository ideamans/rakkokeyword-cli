package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ideamans/rakkokeyword-cli/internal/output"
	"github.com/ideamans/rakkokeyword-cli/internal/rakko"
)

// urlMatchTypes are the site-matching modes shared by several endpoints.
var urlMatchTypes = []string{"url", "forward_url", "domain", "sub_domain"}

// buildTargets assembles the {url, matchType} array the influx endpoints take.
// --targets-json wins outright: it is the only way to give each target its own
// match type.
func buildTargets(urls []string, matchType, targetsJSON string) ([]any, error) {
	if strings.TrimSpace(targetsJSON) != "" {
		var parsed []any
		if err := json.Unmarshal([]byte(targetsJSON), &parsed); err != nil {
			return nil, fmt.Errorf("--targets-json is not a JSON array: %w", err)
		}
		if len(parsed) == 0 {
			return nil, fmt.Errorf("--targets-json is empty")
		}
		return parsed, nil
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("at least one --target is required (or --targets-json)")
	}
	if len(urls) > 20 {
		return nil, fmt.Errorf("too many targets: %d (the API accepts at most 20)", len(urls))
	}
	if err := rakko.Enum("match-type", matchType, urlMatchTypes); err != nil {
		return nil, err
	}
	targets := make([]any, 0, len(urls))
	for _, u := range urls {
		t := map[string]any{"url": u}
		if matchType != "" {
			t["matchType"] = matchType
		}
		targets = append(targets, t)
	}
	return targets, nil
}

// targetFlags are the shared target flags of the influx commands.
type targetFlags struct {
	urls        []string
	matchType   string
	targetsJSON string
}

func (t *targetFlags) addTo(cmd *cobra.Command) {
	f := cmd.Flags()
	f.StringArrayVar(&t.urls, "target", nil, "Domain or URL to investigate; repeat for up to 20 targets")
	f.StringVar(&t.matchType, "match-type", "", "How every --target matches: url / forward_url / domain / sub_domain (API default: sub_domain)")
	f.StringVar(&t.targetsJSON, "targets-json", "", `Targets as raw JSON, for per-target match types: '[{"url":"https://a/","matchType":"url"}]'`)
}

// ── rakkokeyword influx-keywords ────────────────────────────────────────────────────

var (
	influxKeywordsTargets  targetFlags
	influxKeywordsList     listFlags
	influxKeywordsCollapse bool
)

var influxKeywordsCmd = &cobra.Command{
	Use:     "influx-keywords",
	Aliases: []string{"influx-kw"},
	Short:   "Keywords a site or page already earns Google traffic from (4.5 credits)",
	Long: "The keywords a domain or URL ranks for in Google, with its position and\n" +
		"estimated monthly traffic per keyword. Up to 10,000 records.\n\n" +
		"Run it on a competitor to see what they win on, on your own site to see\n" +
		"what you win on, and compare the two for the content gap.\n\n" +
		"Ranks and metrics may be stale; `rakkokeyword search-rank register` re-checks\n" +
		"positions and `rakkokeyword search-volume register` refreshes SEO metrics.\n\n" +
		"Cost: 4.5 credits per request.",
	Example: "  rakkokeyword influx-keywords --target https://example.com/ --match-type sub_domain -n 200\n" +
		"  rakkokeyword influx-keywords --target https://example.com/blog/post --match-type url --sort-by rank --order-by asc",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		targets, err := buildTargets(influxKeywordsTargets.urls, influxKeywordsTargets.matchType, influxKeywordsTargets.targetsJSON)
		if err != nil {
			return err
		}
		body := rakko.Body{"targets": targets}
		body.SetIf(cmd.Flags().Changed("keyword-collapse"), "keywordCollapse", influxKeywordsCollapse)
		if err := influxKeywordsList.applyTo(cmd, body); err != nil {
			return err
		}
		return run(cmd, call{
			req:     rakko.Request{Method: "POST", Path: "/v1/influx-keywords", Body: body},
			credits: "4.5 credits per request",
			out: output.Options{
				ItemsPath: "data.items",
				Caption: []string{"data.summary.totalCount", "data.summary.returnedCount",
					"data.summary.estimatedTraffic", "data.summary.keywordCount"},
				Columns: []string{
					"keyword", "ranking.position", "ranking.estimatedTraffic", "ranking.url",
					"metrics.searchVolume", "metrics.seoDifficulty",
				},
			},
		})
	},
}

func init() {
	rootCmd.AddCommand(influxKeywordsCmd)
	influxKeywordsTargets.addTo(influxKeywordsCmd)
	influxKeywordsCmd.Flags().BoolVar(&influxKeywordsCollapse, "keyword-collapse", false, "Collapse duplicate keywords across targets")
	influxKeywordsList.addTo(influxKeywordsCmd,
		[]string{"keyword", "seoDifficulty", "rank", "searchVolume", "cpc", "competition", "etv"},
		"etv", "desc", "1-10000 (API default: 100)", rakko.InfluxKeywordsFilters)
}

// ── rakkokeyword influx-pages ───────────────────────────────────────────────────────

var (
	influxPagesTargets  targetFlags
	influxPagesList     listFlags
	influxPagesCollapse bool
)

var influxPagesCmd = &cobra.Command{
	Use:     "influx-pages",
	Aliases: []string{"pages"},
	Short:   "Pages of a site that earn the most Google traffic (4.5 credits)",
	Long: "The same data as `rakkokeyword influx-keywords` aggregated per page: total\n" +
		"estimated traffic, traffic value in USD, how many keywords the page ranks\n" +
		"for, and its single best keyword. Up to 10,000 records.\n\n" +
		"A competitor's top pages show which topics already have proven demand.\n" +
		"To see everything one of those pages ranks for, feed its URL back into\n" +
		"`rakkokeyword influx-keywords --match-type url`.\n\n" +
		"Cost: 4.5 credits per request.",
	Example: "  rakkokeyword influx-pages --target https://example.com/ -n 50\n" +
		"  rakkokeyword influx-pages --target https://example.com/ --filter totalEtv.min=100 -f csv",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		targets, err := buildTargets(influxPagesTargets.urls, influxPagesTargets.matchType, influxPagesTargets.targetsJSON)
		if err != nil {
			return err
		}
		body := rakko.Body{"targets": targets}
		body.SetIf(cmd.Flags().Changed("top-keyword-collapse"), "topKeywordCollapse", influxPagesCollapse)
		if err := influxPagesList.applyTo(cmd, body); err != nil {
			return err
		}
		return run(cmd, call{
			req:     rakko.Request{Method: "POST", Path: "/v1/influx-pages", Body: body},
			credits: "4.5 credits per request",
			out: output.Options{
				ItemsPath: "data.items",
				Caption: []string{"data.summary.totalCount", "data.summary.returnedCount",
					"data.summary.estimatedTraffic", "data.summary.keywordCount"},
				Columns: []string{
					"page.url", "page.title", "performance.estimatedTraffic",
					"performance.rankingKeywordCount", "performance.trafficValue",
					"topKeyword.keyword", "topKeyword.position",
				},
			},
		})
	},
}

func init() {
	rootCmd.AddCommand(influxPagesCmd)
	influxPagesTargets.addTo(influxPagesCmd)
	influxPagesCmd.Flags().BoolVar(&influxPagesCollapse, "top-keyword-collapse", false, "Collapse pages that share the same top keyword")
	influxPagesList.addTo(influxPagesCmd,
		[]string{"totalEtv", "totalTrafficValue", "keywordCount"},
		"totalEtv", "desc", "1-10000 (API default: 100)", rakko.InfluxPagesFilters)
}

// ── rakkokeyword competitive ────────────────────────────────────────────────────────

var competitiveList listFlags

var competitiveCmd = &cobra.Command{
	Use:     "competitive <url>",
	Aliases: []string{"competitors"},
	Short:   "Sites whose ranking keywords overlap with a given site (4.5 credits)",
	Long: "Up to 20 sites that rank for the same keywords as the given domain, with\n" +
		"the overlap rate, estimated traffic, traffic value, keyword count and page\n" +
		"count of each.\n\n" +
		"duplicateRate is a fraction in [0,1] — 0.42 means 42% of keywords overlap.\n" +
		"competitorUniqueKeywordCount is the content gap: keywords they have and\n" +
		"the target does not.\n\n" +
		"Cost: 4.5 credits per request.",
	Example: "  rakkokeyword competitive https://example.com/\n" +
		"  rakkokeyword competitive https://example.com/ --sort-by duplicateRate -f json",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		body := rakko.Body{"url": args[0]}
		if err := competitiveList.applyTo(cmd, body); err != nil {
			return err
		}
		return run(cmd, call{
			req:     rakko.Request{Method: "POST", Path: "/v1/competitive", Body: body},
			credits: "4.5 credits per request",
			out: output.Options{
				ItemsPath: "data.items",
				Caption:   []string{"data.summary.totalCount", "data.summary.returnedCount"},
				Columns: []string{
					"site.domain", "metrics.duplicateKeywordCount", "metrics.duplicateRate",
					"metrics.competitorUniqueKeywordCount", "metrics.estimatedTraffic",
					"metrics.keywordCount", "metrics.pageCount",
				},
			},
		})
	},
}

func init() {
	rootCmd.AddCommand(competitiveCmd)
	competitiveList.addTo(competitiveCmd,
		[]string{"duplicate", "duplicateRate", "competitorUnique", "targetUnique", "etv", "keywordCount", "trafficValue", "pageCount"},
		"etv", "desc", "", nil)
}

// ── rakkokeyword bulk-site-research ─────────────────────────────────────────────────

var (
	bulkSiteURLs      []string
	bulkSiteFile      string
	bulkSiteMatchType string
)

var bulkSiteCmd = &cobra.Command{
	Use:     "bulk-site-research [url...]",
	Aliases: []string{"bulk-sites"},
	Short:   "Traffic, keyword and page scale of up to 100 sites at once (0.45 credits per URL)",
	Long: "Current scale and 12-month trend for many sites in one call: estimated\n" +
		"traffic, keyword count, page count, rank distribution and per-page averages.\n\n" +
		"The histories series are indices normalised to 100 at the series maximum —\n" +
		"etvIndex, keywordCountIndex and pageCountIndex — not absolute values. The\n" +
		"metrics block holds the real current numbers. Items come back in the same\n" +
		"order as the URLs given.\n\n" +
		"Requires the STANDARD plan or above. At most 100 URLs.\n\n" +
		"Cost: 0.45 credits per URL, minimum 4.5 credits (10 URLs → 4.5, 100 → 45).",
	Example: "  rakkokeyword bulk-site-research https://a.example/ https://b.example/\n" +
		"  rakkokeyword bulk-site-research --urls-file sites.txt --url-match-type sub_domain -f csv",
	RunE: func(cmd *cobra.Command, args []string) error {
		urls, err := collectValues(append(args, bulkSiteURLs...), bulkSiteFile)
		if err != nil {
			return err
		}
		if len(urls) == 0 {
			return fmt.Errorf("at least one URL is required (as an argument, --url or --urls-file)")
		}
		if len(urls) > 100 {
			return fmt.Errorf("too many URLs: %d (the API accepts at most 100)", len(urls))
		}
		if err := rakko.Enum("url-match-type", bulkSiteMatchType, urlMatchTypes); err != nil {
			return err
		}
		body := rakko.Body{"urls": urls}
		body.SetIf(cmd.Flags().Changed("url-match-type"), "urlMatchType", bulkSiteMatchType)
		return run(cmd, call{
			req:     rakko.Request{Method: "POST", Path: "/v1/bulk-site-research", Body: body},
			credits: fmt.Sprintf("0.45 credits per URL, minimum 4.5 (%d URLs → %.2f credits)", len(urls), maxFloat(4.5, 0.45*float64(len(urls)))),
			out: output.Options{
				ItemsPath: "data.items",
				Caption:   []string{"data.query.urlMatchType", "data.summary.totalCount", "data.summary.returnedCount"},
				Columns: []string{
					"site.target", "metrics.estimatedTraffic", "metrics.estimatedTrafficChangeRate",
					"metrics.keywordCount", "metrics.pageCount", "metrics.trafficValue",
				},
			},
		})
	},
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func init() {
	rootCmd.AddCommand(bulkSiteCmd)
	f := bulkSiteCmd.Flags()
	f.StringArrayVar(&bulkSiteURLs, "url", nil, "URL to research; repeat, or pass URLs as arguments")
	f.StringVar(&bulkSiteFile, "urls-file", "", "File with one URL per line (- for stdin)")
	f.StringVar(&bulkSiteMatchType, "url-match-type", "", "Unit of research: url / forward_url / domain / sub_domain (API default: domain)")
}

// ── rakkokeyword content-search ─────────────────────────────────────────────────────

var (
	contentList        listFlags
	contentTarget      string
	contentAdvanced    bool
	contentCollapseTop bool
)

var contentSearchCmd = &cobra.Command{
	Use:     "content-search <keyword>",
	Aliases: []string{"content"},
	Short:   "Pages whose title, description or top keywords match a keyword (4.5 credits)",
	Long: "Finds web pages related to a keyword and reports their estimated traffic,\n" +
		"traffic value, ranking keyword count and best keyword. Up to 5,000 records.\n\n" +
		"Good for finding places to pitch a guest post or an ad, for competitor\n" +
		"content research, and — with --top-keyword-collapse — for surfacing niche\n" +
		"keywords that weak sites are ranking for unintentionally.\n\n" +
		"Cost: 4.5 credits per request.",
	Example: "  rakkokeyword content-search ラッコ --search-target title -n 100\n" +
		"  rakkokeyword content ラッコ --top-keyword-collapse --filter estimatedTraffic.min=500",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targets := []string{"title", "keyword", "description", "titleAndKeyword", "titleAndKeywordAndDescription"}
		if err := rakko.Enum("search-target", contentTarget, targets); err != nil {
			return err
		}
		body := rakko.Body{"keyword": args[0]}
		body.SetIf(cmd.Flags().Changed("search-target"), "searchTarget", contentTarget)
		body.SetIf(cmd.Flags().Changed("advanced-search"), "isAdvancedSearch", contentAdvanced)
		body.SetIf(cmd.Flags().Changed("top-keyword-collapse"), "topKeywordCollapse", contentCollapseTop)
		if err := contentList.applyTo(cmd, body); err != nil {
			return err
		}
		return run(cmd, call{
			req:     rakko.Request{Method: "POST", Path: "/v1/content-search", Body: body},
			credits: "4.5 credits per request",
			out: output.Options{
				ItemsPath: "data.items",
				Caption:   keywordCaption,
				Columns: []string{
					"page.url", "page.title", "metrics.estimatedTraffic", "metrics.trafficValue",
					"metrics.rankingKeywordCount", "topKeyword.keyword", "topKeyword.position",
				},
			},
		})
	},
}

func init() {
	rootCmd.AddCommand(contentSearchCmd)
	f := contentSearchCmd.Flags()
	f.StringVar(&contentTarget, "search-target", "", "Where the keyword must appear: title / keyword / description / titleAndKeyword /\ntitleAndKeywordAndDescription (API default: titleAndKeywordAndDescription)")
	f.BoolVar(&contentAdvanced, "advanced-search", true, "Morphologically analyse the keyword for better matching (API default: true)")
	f.BoolVar(&contentCollapseTop, "top-keyword-collapse", false, "Keep only one page per top keyword")
	contentList.addTo(contentSearchCmd,
		[]string{"estimatedTraffic", "trafficValue", "rankingKeywordCount"},
		"trafficValue", "desc", "1-5000 (API default: 100)", rakko.ContentSearchFilters)
}

// ── rakkokeyword site-search ────────────────────────────────────────────────────────

var siteSearchList listFlags

var siteSearchCmd = &cobra.Command{
	Use:     "site-search",
	Aliases: []string{"sites"},
	Short:   "Find sites by content, domain or SEO scale (1.5 credits)",
	Long: "Searches whole sites rather than pages, ordered by estimated traffic, and\n" +
		"reports traffic, traffic value, ranking keyword count and page count.\n" +
		"At most 100 records.\n\n" +
		"With a content filter (filter.keyword.includes) the API first takes the\n" +
		"100 most-trafficked related sites and only then applies the other filters,\n" +
		"so filtering cannot page past the first 100 — narrow the content filter\n" +
		"instead. A content filter also adds relatedContent metrics to each record.\n\n" +
		"Cost: 1.5 credits per request.",
	Example: "  rakkokeyword site-search --filter keyword.includes=ラッコ -n 20\n" +
		"  rakkokeyword site-search --filter domain.includes=.jp --filter totalEtv.min=10000 -f json",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		body := rakko.Body{}
		if err := siteSearchList.applyTo(cmd, body); err != nil {
			return err
		}
		return run(cmd, call{
			req:     rakko.Request{Method: "POST", Path: "/v1/site-search", Body: body},
			credits: "1.5 credits per request",
			out: output.Options{
				ItemsPath: "data.items",
				Caption:   []string{"data.summary.totalCount", "data.summary.returnedCount"},
				Columns: []string{
					"site.domain", "site.title", "metrics.estimatedTraffic",
					"metrics.trafficValue", "metrics.rankingKeywordCount", "metrics.pageCount",
				},
			},
		})
	},
}

func init() {
	rootCmd.AddCommand(siteSearchCmd)
	siteSearchList.addTo(siteSearchCmd, nil, "", "", "1-100 (API default: 100)", rakko.SiteSearchFilters)
}

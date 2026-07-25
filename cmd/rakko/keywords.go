package main

import (
	"github.com/spf13/cobra"

	"github.com/ideamans/rakkokeyword-cli/internal/output"
	"github.com/ideamans/rakkokeyword-cli/internal/rakko"
)

// keywordCaption is the summary line above keyword tables.
var keywordCaption = []string{"data.query.keyword", "data.summary.totalCount", "data.summary.returnedCount"}

// seoColumns is the column set every keyword-shaped response shares.
var seoColumns = []string{
	"keyword",
	"metrics.searchVolume",
	"metrics.seoDifficulty",
	"metrics.cpc",
	"metrics.competition",
	"metrics.firstSeenRange",
}

// ── rakko suggest-keywords ───────────────────────────────────────────────────

var suggestModes = []string{
	"google", "bing", "youtube", "googleVideo", "amazon", "rakuten", "googleShopping", "googleImage",
}

var (
	suggestList     listFlags
	suggestModeList []string
	suggestIncrease bool
)

var suggestCmd = &cobra.Command{
	Use:     "suggest-keywords <keyword>",
	Aliases: []string{"suggest"},
	Short:   "Search-engine suggestions for a keyword, with SEO metrics (1.5 credits)",
	Long: "Autocomplete suggestions for a keyword from Google, Bing, YouTube, Amazon,\n" +
		"Rakuten and others, with monthly search volume, SEO difficulty, CPC and\n" +
		"competition attached.\n\n" +
		"This is the usual first step of keyword research: it shows the compound\n" +
		"queries real users type. About 1,000 suggestions are available normally and\n" +
		"about 10,000 with --increase-keyword.\n\n" +
		"The attached SEO metrics may be stale. When they matter, feed the keywords\n" +
		"into `rakko search-volume register` for fresh figures.\n\n" +
		"Cost: 1.5 credits per request.",
	Example: "  rakko suggest-keywords ラッコ --modes google,bing -n 50\n" +
		"  rakko suggest ラッコ --increase-keyword --filter searchVolume.min=100 -f json",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := rakko.EnumList("modes", suggestModeList, suggestModes); err != nil {
			return err
		}
		body := rakko.Body{"keyword": args[0]}
		body.SetIf(len(suggestModeList) > 0, "modes", suggestModeList)
		body.SetIf(cmd.Flags().Changed("increase-keyword"), "increaseKeyword", suggestIncrease)
		if err := suggestList.applyTo(cmd, body); err != nil {
			return err
		}
		return run(cmd, call{
			req:     rakko.Request{Method: "POST", Path: "/v1/suggest-keywords", Body: body},
			credits: "1.5 credits per request",
			out: output.Options{
				ItemsPath: "data.items",
				Caption:   keywordCaption,
				Columns:   append([]string{"keyword", "suggestClass"}, seoColumns[1:]...),
			},
		})
	},
}

func init() {
	rootCmd.AddCommand(suggestCmd)
	f := suggestCmd.Flags()
	f.StringSliceVar(&suggestModeList, "modes", nil, "Search engines to pull suggestions from, comma-separated:\n"+
		"google / bing / youtube / googleVideo / amazon / rakuten / googleShopping / googleImage (API default: google)")
	f.BoolVar(&suggestIncrease, "increase-keyword", false, "Fetch the extended suggestion set (~10,000 instead of ~1,000)")
	suggestList.addTo(suggestCmd,
		[]string{"keyword", "suggestClass", "seoDifficulty", "searchVolume", "cpc", "competition", "firstSeenRange"},
		"searchVolume", "desc", "any positive integer (API default: all results)", rakko.SuggestFilters)
}

// ── rakko related-keywords ───────────────────────────────────────────────────

var (
	relatedList      listFlags
	relatedMatchType string
)

var relatedCmd = &cobra.Command{
	Use:     "related-keywords <keyword>",
	Aliases: []string{"related"},
	Short:   "Keywords from the rakko database matching a keyword, up to 25,000 (1.5 credits)",
	Long: "Bulk keyword harvesting: every keyword in the rakkokeyword database that\n" +
		"matches the given one, with SEO metrics, up to 25,000 records.\n\n" +
		"Reach for `rakko suggest-keywords` first — it reflects real search-engine\n" +
		"suggestions. Use this when you need volume beyond what suggestions give.\n\n" +
		"Cost: 1.5 credits per request.",
	Example: "  rakko related-keywords ラッコ --match-type phraseMatch -n 1000\n" +
		"  rakko related ラッコ --filter keyword.notIncludes=グッズ -f csv > keywords.csv",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		matchTypes := []string{"partialMatch", "phraseMatch", "prefixMatch", "suffixMatch", "wordMatch"}
		if err := rakko.Enum("match-type", relatedMatchType, matchTypes); err != nil {
			return err
		}
		body := rakko.Body{"keyword": args[0]}
		body.SetIf(cmd.Flags().Changed("match-type"), "matchType", relatedMatchType)
		if err := relatedList.applyTo(cmd, body); err != nil {
			return err
		}
		return run(cmd, call{
			req:     rakko.Request{Method: "POST", Path: "/v1/related-keywords", Body: body},
			credits: "1.5 credits per request",
			out: output.Options{
				ItemsPath: "data.items",
				Caption:   keywordCaption,
				Columns:   seoColumns,
			},
		})
	},
}

func init() {
	rootCmd.AddCommand(relatedCmd)
	relatedCmd.Flags().StringVar(&relatedMatchType, "match-type", "",
		"How the keyword must match: partialMatch / phraseMatch / prefixMatch / suffixMatch / wordMatch (API default: partialMatch)")
	relatedList.addTo(relatedCmd,
		[]string{"seoDifficulty", "searchVolume", "cpc", "competition", "firstSeenRange"},
		"searchVolume", "desc", "1-25000 (API default: 1000)", rakko.RelatedFilters)
}

// ── rakko other-keywords ─────────────────────────────────────────────────────

var otherList listFlags

var otherCmd = &cobra.Command{
	Use:     "other-keywords <keyword>",
	Aliases: []string{"other", "lsi", "paa"},
	Short:   "LSI keywords and People-Also-Ask questions, recursively (22.5 credits)",
	Long: "What Google thinks someone searching this keyword will look for next\n" +
		"(\"People also search for\", LSI) and what they are wondering about\n" +
		"(\"People also ask\", PAA), gathered recursively up to two levels.\n\n" +
		"The importance field (high / medium / low) counts how often an entry\n" +
		"reappeared during the recursion — high means Google surfaces it broadly.\n\n" +
		"This is the most expensive per-request command in the CLI.\n\n" +
		"Cost: 22.5 credits per request.",
	Example: "  rakko other-keywords ラッコ\n" +
		"  rakko other ラッコ -f json | jq '.data.items[] | select(.type==\"paa\")'",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		body := rakko.Body{"keyword": args[0]}
		if err := otherList.applyTo(cmd, body); err != nil {
			return err
		}
		return run(cmd, call{
			req:     rakko.Request{Method: "POST", Path: "/v1/other-keywords", Body: body},
			credits: "22.5 credits per request",
			out: output.Options{
				ItemsPath: "data.items",
				Caption:   []string{"data.query.keyword", "data.summary.lsiCount", "data.summary.paaCount"},
				Columns: []string{
					"type", "keyword", "question", "importance", "sourceKeyword",
					"metrics.searchVolume", "metrics.seoDifficulty",
				},
			},
		})
	},
}

func init() {
	rootCmd.AddCommand(otherCmd)
	otherList.addTo(otherCmd,
		[]string{"importance", "seoDifficulty", "searchVolume", "cpc", "competition", "firstSeenRange"},
		"importance", "desc", "", nil)
}

// ── rakko question-search ────────────────────────────────────────────────────

var questionLimit int

var questionCmd = &cobra.Command{
	Use:     "question-search <keyword>",
	Aliases: []string{"questions"},
	Short:   "Frequently asked questions containing a keyword, by frequency (3 credits)",
	Long: "Questions from the rakkokeyword database that contain the keyword, ordered\n" +
		"by how often they occur. Up to 200 records.\n\n" +
		"Useful for FAQ and Q&A content, and for AIO / GEO / LLMO work: these are\n" +
		"the phrasings people are likely to type into an AI assistant.\n\n" +
		"For the questions Google itself shows on a SERP, use `rakko other-keywords`.\n\n" +
		"Cost: 3 credits per request.",
	Example: "  rakko question-search ラッコ -n 50\n" +
		"  rakko questions ラッコ -f jsonl",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		body := rakko.Body{"keyword": args[0]}
		body.SetIf(cmd.Flags().Changed("limit"), "limit", questionLimit)
		return run(cmd, call{
			req:     rakko.Request{Method: "POST", Path: "/v1/question-search", Body: body},
			credits: "3 credits per request",
			out: output.Options{
				ItemsPath: "data.items",
				Caption:   keywordCaption,
				Columns:   []string{"question"},
			},
		})
	},
}

func init() {
	rootCmd.AddCommand(questionCmd)
	questionCmd.Flags().IntVarP(&questionLimit, "limit", "n", 0, "Maximum questions to return, 1-200 (API default: 100)")
}

// ── rakko ranking-keywords ───────────────────────────────────────────────────

var (
	rankingList        listFlags
	rankingSearchTop   int
	rankingSearchRange int
)

var rankingCmd = &cobra.Command{
	Use:     "ranking-keywords <keyword>",
	Aliases: []string{"ranking", "co-ranking"},
	Short:   "Keywords the pages ranking for this keyword also rank for (4.5 credits)",
	Long: "Takes the pages that rank highly for the keyword and reports the other\n" +
		"keywords those same pages rank for. Up to 5,000 records with SEO metrics.\n\n" +
		"relevance (1-100) is how much the two result sets overlap. High-relevance\n" +
		"keywords share search intent and can usually be targeted by one article;\n" +
		"low-relevance ones deserve their own.\n\n" +
		"Narrow --search-top and --search-range for closer intent, widen them to\n" +
		"discover keywords further afield.\n\n" +
		"Cost: 4.5 credits per request.",
	Example: "  rakko ranking-keywords ラッコ --search-top 10 --search-range 20\n" +
		"  rakko ranking ラッコ --filter relevance.min=50 -n 200",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("search-top") {
			if err := allowedInt("search-top", rankingSearchTop, []int{3, 5, 10, 20, 30, 50}); err != nil {
				return err
			}
		}
		if cmd.Flags().Changed("search-range") {
			if err := allowedInt("search-range", rankingSearchRange, []int{10, 20, 30, 50, 100}); err != nil {
				return err
			}
		}
		body := rakko.Body{"keyword": args[0]}
		body.SetIf(cmd.Flags().Changed("search-top"), "searchTop", rankingSearchTop)
		body.SetIf(cmd.Flags().Changed("search-range"), "searchRange", rankingSearchRange)
		if err := rankingList.applyTo(cmd, body); err != nil {
			return err
		}
		return run(cmd, call{
			req:     rakko.Request{Method: "POST", Path: "/v1/ranking-keywords", Body: body},
			credits: "4.5 credits per request",
			out: output.Options{
				ItemsPath: "data.items",
				Caption:   keywordCaption,
				Columns: []string{
					"keyword", "metrics.relevance", "metrics.searchVolume",
					"metrics.seoDifficulty", "metrics.cpc", "metrics.competition",
				},
			},
		})
	},
}

func init() {
	rootCmd.AddCommand(rankingCmd)
	f := rankingCmd.Flags()
	f.IntVar(&rankingSearchTop, "search-top", 0, "How many top-ranking pages to inspect: 3 / 5 / 10 / 20 / 30 / 50 (API default: 20)")
	f.IntVar(&rankingSearchRange, "search-range", 0, "Rank cut-off for the keywords those pages rank for: 10 / 20 / 30 / 50 / 100 (API default: 50)")
	rankingList.addTo(rankingCmd,
		[]string{"seoDifficulty", "searchVolume", "cpc", "competition", "relevance"},
		"relevance", "desc", "1-5000 (API default: 500)", rakko.RankingFilters)
}

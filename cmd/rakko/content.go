package main

import (
	"github.com/spf13/cobra"

	"github.com/ideamans/rakkokeyword-cli/internal/output"
	"github.com/ideamans/rakkokeyword-cli/internal/rakko"
)

// ── rakko headline ───────────────────────────────────────────────────────────

var (
	headlineList          listFlags
	headlineLessHeadlines bool
	headlineLessChars     bool
	headlineLevels        = map[string]*bool{}
)

var headlineCmd = &cobra.Command{
	Use:     "headline <keyword>",
	Aliases: []string{"headlines"},
	Short:   "Headings (h1-h6) of the pages ranking for a keyword (3 credits)",
	Long: "Extracts the headings of the top Google results for a keyword, plus each\n" +
		"page's character and heading count and the averages across them.\n\n" +
		"Topics that recur across the top pages are the ones Google's users are\n" +
		"assumed to need covered, which makes this the step before drafting a title\n" +
		"and outline. Pair it with `rakko co-occurrence` for the vocabulary.\n\n" +
		"Only h1-h4 are collected by default; add --h5 --h6 for the rest.\n\n" +
		"Cost: 3 credits per request.",
	Example: "  rakko headline ラッコ\n" +
		"  rakko headline ラッコ --less-characters -f json | jq '.data.items[].headlines'",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		body := rakko.Body{"keyword": args[0]}
		body.SetIf(cmd.Flags().Changed("less-headlines"), "lessHeadlines", headlineLessHeadlines)
		body.SetIf(cmd.Flags().Changed("less-characters"), "lessCharacters", headlineLessChars)
		for _, level := range []string{"h1", "h2", "h3", "h4", "h5", "h6"} {
			body.SetIf(cmd.Flags().Changed(level), level, *headlineLevels[level])
		}
		if err := headlineList.applyTo(cmd, body); err != nil {
			return err
		}
		return run(cmd, call{
			req:     rakko.Request{Method: "POST", Path: "/v1/headline", Body: body},
			credits: "3 credits per request",
			out: output.Options{
				ItemsPath: "data.items",
				Caption: []string{"data.query.keyword", "data.summary.returnedCount",
					"data.summary.averageHeadlineCount", "data.summary.averageWordCount"},
				Columns: []string{
					"metrics.position", "page.title", "page.url",
					"metrics.headlineCount", "metrics.wordCount",
				},
			},
		})
	},
}

func init() {
	rootCmd.AddCommand(headlineCmd)
	f := headlineCmd.Flags()
	f.BoolVar(&headlineLessHeadlines, "less-headlines", false, "Exclude pages with fewer than 5 headings")
	f.BoolVar(&headlineLessChars, "less-characters", false, "Exclude pages with fewer than 1,000 characters")
	for _, level := range []string{"h1", "h2", "h3", "h4"} {
		v := new(bool)
		headlineLevels[level] = v
		f.BoolVar(v, level, true, "Include <"+level+"> headings (API default: true)")
	}
	for _, level := range []string{"h5", "h6"} {
		v := new(bool)
		headlineLevels[level] = v
		f.BoolVar(v, level, false, "Include <"+level+"> headings (API default: false)")
	}
	headlineList.addTo(headlineCmd,
		[]string{"position", "title", "headlineCount", "wordCount"},
		"position", "asc", "1-20 (API default: 20)", nil)
}

// ── rakko co-occurrence ──────────────────────────────────────────────────────

var (
	cooccurrenceList    listFlags
	cooccurrenceDetails bool
)

var cooccurrenceCmd = &cobra.Command{
	Use:     "co-occurrence <keyword>",
	Aliases: []string{"cooc", "cooccurrence"},
	Short:   "Words that recur across the pages ranking for a keyword (3 credits)",
	Long: "The vocabulary of the top Google results for a keyword: how often each word\n" +
		"appears in body text, titles and headings, and on how many of the ranking\n" +
		"sites.\n\n" +
		"Words shared by most top pages are the ones an article on this topic is\n" +
		"expected to contain. siteCountTotal — how many ranking sites use the word at\n" +
		"all — is the most robust signal; a single verbose page can inflate the\n" +
		"occurrence counts on its own.\n\n" +
		"--details=false drops the per-page breakdown and makes the response much\n" +
		"smaller.\n\n" +
		"Cost: 3 credits per request.",
	Example: "  rakko co-occurrence ラッコ -n 30\n" +
		"  rakko cooc ラッコ --details=false -f csv",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		body := rakko.Body{"keyword": args[0]}
		body.SetIf(cmd.Flags().Changed("details"), "getDetails", cooccurrenceDetails)
		if err := cooccurrenceList.applyTo(cmd, body); err != nil {
			return err
		}
		return run(cmd, call{
			req:     rakko.Request{Method: "POST", Path: "/v1/co-occurrence", Body: body},
			credits: "3 credits per request",
			out: output.Options{
				ItemsPath: "data.items",
				Caption:   keywordCaption,
				Columns: []string{
					"word", "metrics.siteCountTotal", "metrics.siteCountHeading",
					"metrics.occurrencePageCount", "metrics.occurrenceTitleCount", "metrics.occurrenceHeadingCount",
				},
			},
		})
	},
}

func init() {
	rootCmd.AddCommand(cooccurrenceCmd)
	cooccurrenceCmd.Flags().BoolVar(&cooccurrenceDetails, "details", true, "Include the per-page breakdown for every word (API default: true)")
	cooccurrenceList.addTo(cooccurrenceCmd,
		[]string{"word", "occurrencePageCount", "occurrenceTitleCount", "occurrenceHeadingCount", "siteCountTotal", "siteCountHeading"},
		"siteCountTotal", "desc", "any positive integer (API default: all results)", nil)
}

// ── rakko metadata ───────────────────────────────────────────────────────────

var metadataCmd = &cobra.Command{
	Use:   "metadata",
	Short: "Region and language names accepted by --location and --language (free)",
}

var (
	metadataLocationName string
	metadataCountryCode  string
	metadataLimit        int
)

var metadataLocationsCmd = &cobra.Command{
	Use:   "locations",
	Short: "List region names for --location (free, no API key needed)",
	Long: "The region names `rakko search-volume register --location` and\n" +
		"`rakko search-rank register --location` accept.\n\n" +
		"Unfiltered the list is country-level only. Give --location-name or\n" +
		"--country-code and city-level regions appear too; those are written as\n" +
		"\"City,Region,Country\" (e.g. Shibuya,Tokyo,Japan). Intermediate levels on\n" +
		"their own — a prefecture with no city — are not supported.\n\n" +
		"Cost: free. This endpoint needs no API key.",
	Example: "  rakko metadata locations --country-code JP\n" +
		"  rakko metadata locations --location-name Tokyo",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		q := queryValues()
		q.setIf(cmd, "location-name", "locationName", metadataLocationName)
		q.setIf(cmd, "country-code", "countryCode", metadataCountryCode)
		q.setIfInt(cmd, "limit", "limit", metadataLimit)
		return run(cmd, call{
			req:     rakko.Request{Method: "GET", Path: "/v1/metadata/locations", Query: q.values, NoAuth: true},
			credits: "free",
			out: output.Options{
				ItemsPath: "data.locations",
				Columns:   []string{"name", "countryIsoCode"},
			},
		})
	},
}

var metadataLanguagesCmd = &cobra.Command{
	Use:   "languages",
	Short: "List language names for --language (free, no API key needed)",
	Long: "The language names `rakko search-volume register --language` and\n" +
		"`rakko search-rank register --language` accept. Use the value verbatim\n" +
		"(e.g. Japanese).\n\n" +
		"Cost: free. This endpoint needs no API key.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return run(cmd, call{
			req:     rakko.Request{Method: "GET", Path: "/v1/metadata/languages", NoAuth: true},
			credits: "free",
			out: output.Options{
				ItemsPath: "data.languages",
				Columns:   []string{"name"},
			},
		})
	},
}

func init() {
	rootCmd.AddCommand(metadataCmd)
	metadataCmd.AddCommand(metadataLocationsCmd)
	metadataCmd.AddCommand(metadataLanguagesCmd)
	f := metadataLocationsCmd.Flags()
	f.StringVar(&metadataLocationName, "location-name", "", "Filter by region name (substring, case-insensitive); also reveals city-level regions")
	f.StringVar(&metadataCountryCode, "country-code", "", "Filter by ISO 3166-1 alpha-2 country code (e.g. JP); also reveals city-level regions")
	f.IntVarP(&metadataLimit, "limit", "n", 0, "Maximum records to return (API default: all)")
}

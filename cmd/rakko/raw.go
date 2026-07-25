package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ideamans/rakkokeyword-cli/internal/output"
	"github.com/ideamans/rakkokeyword-cli/internal/rakko"
)

var (
	rawData      string
	rawQuery     []string
	rawItemsPath string
)

var rawCmd = &cobra.Command{
	Use:   "raw <METHOD> <path>",
	Short: "Call any API endpoint directly with a JSON body (cost: whatever that endpoint costs)",
	Long: "The escape hatch. Sends a request to any path of the rakkokeyword API and\n" +
		"prints the response, so a parameter this CLI has not wrapped is never a\n" +
		"dead end.\n\n" +
		"The body comes from --data, from @file, or from stdin when --data is \"-\".\n" +
		"Authentication, retries, timeouts and output formatting work as they do\n" +
		"everywhere else.\n\n" +
		"It charges the same credits as the endpoint it calls; --dry-run still shows\n" +
		"the request without sending it. The endpoint list is in `rakko llm` and at\n" +
		"https://api.rakkokeyword.com/api-docs.json.",
	Example: `  rakko raw POST /v1/suggest-keywords --data '{"keyword":"ラッコ","modes":["google"]}'` + "\n" +
		"  rakko raw GET /v1/metadata/locations --query countryCode=JP\n" +
		"  rakko raw POST /v1/co-occurrence --data @body.json --items data.items -f csv",
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		method := strings.ToUpper(args[0])
		path := args[1]
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}

		body, err := rawBody(rawData)
		if err != nil {
			return err
		}

		query := url.Values{}
		for _, q := range rawQuery {
			key, value, ok := strings.Cut(q, "=")
			if !ok {
				return fmt.Errorf("invalid --query %q: expected key=value", q)
			}
			query.Add(key, value)
		}

		return run(cmd, call{
			req:     rakko.Request{Method: method, Path: path, Query: query, Body: body},
			credits: "whatever this endpoint costs — see `rakko llm`",
			out:     output.Options{ItemsPath: rawItemsPath},
		})
	},
}

// rawBody reads the request body from the flag, a file or stdin, and validates
// that it is JSON before spending a credit on a body the API will reject.
func rawBody(spec string) (any, error) {
	if spec == "" {
		return nil, nil
	}
	var data []byte
	var err error
	switch {
	case spec == "-":
		data, err = io.ReadAll(os.Stdin)
	case strings.HasPrefix(spec, "@"):
		data, err = os.ReadFile(strings.TrimPrefix(spec, "@"))
	default:
		data = []byte(spec)
	}
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, nil
	}
	var parsed any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("--data is not valid JSON: %w", err)
	}
	return parsed, nil
}

func init() {
	rootCmd.AddCommand(rawCmd)
	f := rawCmd.Flags()
	f.StringVarP(&rawData, "data", "d", "", `JSON request body: inline, @file, or - for stdin`)
	f.StringArrayVar(&rawQuery, "query", nil, "Query parameter as key=value; repeatable")
	f.StringVar(&rawItemsPath, "items", "data.items", "Dotted path to the array to tabulate for table, csv and jsonl output")
}

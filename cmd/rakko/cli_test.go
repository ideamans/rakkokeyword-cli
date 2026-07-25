package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// runCLI executes the real command tree against a stub API and returns stdout.
// Flag state is package-global (cobra commands are built in init), so each test
// resets the flags it touches through resetFlags.
func runCLI(t *testing.T, handler http.HandlerFunc, args ...string) (string, error) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	t.Setenv("RAKKOKEYWORD_API_BASE", server.URL)
	t.Setenv("RAKKOKEYWORD_API_KEY", "test-key")
	t.Setenv("RAKKOKEYWORD_CLI_HOME", t.TempDir()) // never read the developer's own config

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs(args)
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
		resetFlags(rootCmd)
	})

	err := rootCmd.Execute()
	return stdout.String(), err
}

// resetFlags restores every flag in the tree to its default so one test's
// flags cannot leak into the next.
func resetFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Changed = false
		// Slice flags must be emptied through Replace: Set on a slice appends,
		// so restoring the "[]" default string would leave one bogus element.
		if slice, ok := f.Value.(pflag.SliceValue); ok {
			_ = slice.Replace(nil)
			return
		}
		_ = f.Value.Set(f.DefValue)
	})
	for _, sub := range cmd.Commands() {
		resetFlags(sub)
	}
}

func jsonResponse(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, body)
	}
}

const suggestBody = `{"result":true,"meta":{"consumedCredit":1.5},"data":{"query":{"keyword":"ラッコ"},` +
	`"summary":{"totalCount":2,"returnedCount":2},"items":[` +
	`{"keyword":"ラッコ 水族館","suggestClass":"＋","metrics":{"searchVolume":18100,"seoDifficulty":36}},` +
	`{"keyword":"ラッコ グッズ","suggestClass":"＋","metrics":{"searchVolume":1600,"seoDifficulty":21}}]},"errors":[]}`

func TestSuggestSendsBodyAndRendersTable(t *testing.T) {
	var got map[string]any
	out, err := runCLI(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = io.WriteString(w, suggestBody)
	}, "suggest-keywords", "ラッコ", "--modes", "google,bing", "--increase-keyword",
		"--filter", "searchVolume.min=100", "-n", "50")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if got["keyword"] != "ラッコ" {
		t.Errorf("keyword = %v", got["keyword"])
	}
	if got["increaseKeyword"] != true {
		t.Errorf("increaseKeyword = %v", got["increaseKeyword"])
	}
	if got["limit"] != float64(50) {
		t.Errorf("limit = %v", got["limit"])
	}
	modes, _ := got["modes"].([]any)
	if len(modes) != 2 || modes[0] != "google" {
		t.Errorf("modes = %v", got["modes"])
	}
	filter, _ := got["filter"].(map[string]any)
	volume, _ := filter["searchVolume"].(map[string]any)
	if volume["min"] != float64(100) {
		t.Errorf("filter = %v", got["filter"])
	}
	if !strings.Contains(out, "ラッコ 水族館") || !strings.Contains(out, "totalCount=2") {
		t.Errorf("unexpected table:\n%s", out)
	}
}

func TestUnsetFlagsStayOutOfTheBody(t *testing.T) {
	var got map[string]any
	if _, err := runCLI(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = io.WriteString(w, suggestBody)
	}, "suggest-keywords", "ラッコ"); err != nil {
		t.Fatalf("execute: %v", err)
	}
	// Only the keyword should be sent: everything else is the API's default,
	// and mirroring defaults locally would freeze them at today's values.
	if len(got) != 1 {
		t.Errorf("body should carry only the keyword, got %v", got)
	}
}

func TestDryRunSendsNothing(t *testing.T) {
	called := false
	out, err := runCLI(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
	}, "other-keywords", "ラッコ", "--dry-run")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if called {
		t.Error("--dry-run must not reach the API — it exists so an agent can price a call first")
	}
	var preview struct {
		Method, URL, Cost string
		Body              map[string]any
	}
	if err := json.Unmarshal([]byte(out), &preview); err != nil {
		t.Fatalf("--dry-run should print JSON, got %s", out)
	}
	if preview.Method != "POST" || !strings.HasSuffix(preview.URL, "/v1/other-keywords") {
		t.Errorf("preview = %+v", preview)
	}
	if !strings.Contains(preview.Cost, "22.5") {
		t.Errorf("preview should state the cost, got %q", preview.Cost)
	}
}

func TestAPIErrorSurfacesTheServerMessage(t *testing.T) {
	_, err := runCLI(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = io.WriteString(w, `{"result":false,"errors":["Insufficient credit."]}`)
	}, "question-search", "ラッコ")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "Insufficient credit") || !strings.Contains(err.Error(), "credit") {
		t.Errorf("error should carry the API's own message: %v", err)
	}
}

func TestSearchVolumeRegisterWaitPollsThenFetchesResults(t *testing.T) {
	var seen []string
	statusChecks := 0
	out, err := runCLI(t, func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method+" "+r.URL.Path)
		switch {
		case r.URL.Path == "/v1/search-volume":
			_, _ = io.WriteString(w, `{"result":true,"meta":{"consumedCredit":15},"data":{"requestId":1234567},"errors":[]}`)
		case strings.HasSuffix(r.URL.Path, "/status"):
			statusChecks++
			done := statusChecks > 1
			_, _ = io.WriteString(w, `{"result":true,"data":{"isCompleted":`+boolJSON(done)+`,"statuses":{"searchVolume":"processing"}},"errors":[]}`)
		case strings.HasSuffix(r.URL.Path, "/results"):
			_, _ = io.WriteString(w, `{"result":true,"data":{"query":{"requestId":1234567},"summary":{"totalCount":1,"returnedCount":1},`+
				`"items":[{"keyword":"ラッコ","metrics":{"searchVolume":90500}}]},"errors":[]}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}, "search-volume", "register", "ラッコ", "--wait", "--poll-interval", "1ms", "-n", "10")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := []string{
		"POST /v1/search-volume",
		"GET /v1/search-volume/1234567/status",
		"GET /v1/search-volume/1234567/status",
		"POST /v1/search-volume/1234567/results",
	}
	if strings.Join(seen, "|") != strings.Join(want, "|") {
		t.Errorf("call sequence\n got %v\nwant %v", seen, want)
	}
	if !strings.Contains(out, "90500") {
		t.Errorf("results were not printed:\n%s", out)
	}
}

func TestSearchRankRegisterRequiresURLs(t *testing.T) {
	_, err := runCLI(t, jsonResponse(`{}`), "search-rank", "register", "ラッコ")
	if err == nil || !strings.Contains(err.Error(), "--url") {
		t.Errorf("expected a message naming the missing flag, got %v", err)
	}
}

func TestInfluxKeywordsBuildsTargets(t *testing.T) {
	var got map[string]any
	if _, err := runCLI(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = io.WriteString(w, `{"result":true,"data":{"summary":{},"items":[]},"errors":[]}`)
	}, "influx-keywords", "--target", "https://example.com/", "--match-type", "url"); err != nil {
		t.Fatalf("execute: %v", err)
	}
	targets, _ := got["targets"].([]any)
	if len(targets) != 1 {
		t.Fatalf("targets = %v", got["targets"])
	}
	first, _ := targets[0].(map[string]any)
	if first["url"] != "https://example.com/" || first["matchType"] != "url" {
		t.Errorf("target = %v", first)
	}
}

func TestRawPostsArbitraryJSON(t *testing.T) {
	var got map[string]any
	if _, err := runCLI(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		if q := r.URL.Query().Get("countryCode"); q != "JP" {
			t.Errorf("query lost: %q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, `{"result":true,"data":{"items":[]},"errors":[]}`)
	}, "raw", "post", "/v1/anything", "--data", `{"future":"parameter"}`, "--query", "countryCode=JP"); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got["future"] != "parameter" {
		t.Errorf("body = %v", got)
	}
}

func TestRawRejectsMalformedJSONBeforeSpending(t *testing.T) {
	called := false
	_, err := runCLI(t, func(w http.ResponseWriter, _ *http.Request) { called = true },
		"raw", "POST", "/v1/suggest-keywords", "--data", "{not json}")
	if err == nil {
		t.Fatal("expected an error")
	}
	if called {
		t.Error("a malformed body must not be charged for")
	}
}

func boolJSON(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

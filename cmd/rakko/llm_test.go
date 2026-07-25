package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ideamans/go-llm-cli-kit/llmcmd"
)

// TestLLMReferenceRenders checks that the embedded reference is complete: an
// agent that reads it must find the cost rules, the response shapes and the
// command catalog, all of which live in different chapters.
func TestLLMReferenceRenders(t *testing.T) {
	md, err := llmcmd.Render(llmConfig(), "markdown")
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}
	for _, want := range []string{
		"# rakko — reference for AI agents",
		"# Metrics and how to read them",
		"# JSON output schemas",
		"# Limits, costs and traps",
		"# Command catalog",
		"RAKKOKEYWORD_API_KEY",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("reference is missing %q", want)
		}
	}

	out, err := llmcmd.Render(llmConfig(), "json")
	if err != nil {
		t.Fatalf("render json: %v", err)
	}
	var sections []struct{ File, Title, Body string }
	if err := json.Unmarshal([]byte(out), &sections); err != nil {
		t.Fatalf("json output is not an array of sections: %v", err)
	}
	if len(sections) < 5 {
		t.Errorf("want every chapter, got %d", len(sections))
	}
}

// TestCatalogCoversEveryEndpointCommand guards against a command that exists in
// the tree but never reaches the generated catalog — the file agents read to
// discover what this CLI can do.
func TestCatalogCoversEveryEndpointCommand(t *testing.T) {
	md, err := llmcmd.Render(llmConfig(), "markdown")
	if err != nil {
		t.Fatal(err)
	}
	for _, cmd := range []string{
		"rakko suggest-keywords", "rakko related-keywords", "rakko other-keywords",
		"rakko question-search", "rakko ranking-keywords",
		"rakko search-volume register", "rakko search-volume histories",
		"rakko search-volume status", "rakko search-volume results",
		"rakko search-rank register", "rakko search-rank histories",
		"rakko search-rank status", "rakko search-rank results",
		"rakko metadata locations", "rakko metadata languages",
		"rakko influx-keywords", "rakko influx-pages", "rakko competitive",
		"rakko bulk-site-research", "rakko content-search", "rakko headline",
		"rakko co-occurrence", "rakko site-search", "rakko raw",
	} {
		if !strings.Contains(md, "`"+cmd+"`") {
			t.Errorf("the catalog does not document %q — run `go generate ./...`", cmd)
		}
	}
}

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
		"# rakkokeyword — reference for AI agents",
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
		"rakkokeyword suggest-keywords", "rakkokeyword related-keywords", "rakkokeyword other-keywords",
		"rakkokeyword question-search", "rakkokeyword ranking-keywords",
		"rakkokeyword search-volume register", "rakkokeyword search-volume histories",
		"rakkokeyword search-volume status", "rakkokeyword search-volume results",
		"rakkokeyword search-rank register", "rakkokeyword search-rank histories",
		"rakkokeyword search-rank status", "rakkokeyword search-rank results",
		"rakkokeyword metadata locations", "rakkokeyword metadata languages",
		"rakkokeyword influx-keywords", "rakkokeyword influx-pages", "rakkokeyword competitive",
		"rakkokeyword bulk-site-research", "rakkokeyword content-search", "rakkokeyword headline",
		"rakkokeyword co-occurrence", "rakkokeyword site-search", "rakkokeyword raw",
	} {
		if !strings.Contains(md, "`"+cmd+"`") {
			t.Errorf("the catalog does not document %q — run `go generate ./...`", cmd)
		}
	}
}

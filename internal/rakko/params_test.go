package rakko

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseFiltersNestsPaths(t *testing.T) {
	got, err := SuggestFilters.ParseFilters([]string{
		"searchVolume.min=100",
		"searchVolume.max=10000",
		"cpc.min=0.5",
		"firstSeenRange.include=last_30_days",
	})
	if err != nil {
		t.Fatalf("ParseFilters: %v", err)
	}
	data, _ := json.Marshal(got)
	want := `{"cpc":{"min":0.5},"firstSeenRange":{"include":"last_30_days"},"searchVolume":{"max":10000,"min":100}}`
	if string(data) != want {
		t.Errorf("filter JSON\n got %s\nwant %s", data, want)
	}
}

func TestParseFiltersAppendsRepeatedLists(t *testing.T) {
	got, err := SuggestFilters.ParseFilters([]string{
		"keyword.includes=水族館",
		"keyword.includes=動物園,ペット",
		"suggestClass=0,1",
	})
	if err != nil {
		t.Fatalf("ParseFilters: %v", err)
	}
	data, _ := json.Marshal(got)
	want := `{"keyword":{"includes":["水族館","動物園","ペット"]},"suggestClass":[0,1]}`
	if string(data) != want {
		t.Errorf("filter JSON\n got %s\nwant %s", data, want)
	}
}

func TestParseFiltersRejectsUnknownPath(t *testing.T) {
	// The path exists on another endpoint but not on this one; catching it
	// locally is the difference between a clear message and a bare 400.
	_, err := RelatedFilters.ParseFilters([]string{"relevance.min=10"})
	if err == nil {
		t.Fatal("expected an error for a filter path this endpoint does not accept")
	}
	if !strings.Contains(err.Error(), "unknown filter path") {
		t.Errorf("unhelpful error: %v", err)
	}
}

func TestParseFiltersRejectsBadValues(t *testing.T) {
	cases := []struct{ name, assignment string }{
		{"not a number", "searchVolume.min=lots"},
		{"not an enum", "firstSeenRange.include=yesterday"},
		{"no equals sign", "searchVolume.min"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := SuggestFilters.ParseFilters([]string{c.assignment}); err == nil {
				t.Errorf("expected an error for %q", c.assignment)
			}
		})
	}
}

func TestMergeFilterJSONOverridesFlags(t *testing.T) {
	base, err := SuggestFilters.ParseFilters([]string{"searchVolume.min=100"})
	if err != nil {
		t.Fatal(err)
	}
	merged, err := MergeFilterJSON(base, `{"searchVolume":{"min":500}}`)
	if err != nil {
		t.Fatalf("MergeFilterJSON: %v", err)
	}
	data, _ := json.Marshal(merged)
	if string(data) != `{"searchVolume":{"min":500}}` {
		t.Errorf("raw JSON should win, got %s", data)
	}
}

func TestBodySetIf(t *testing.T) {
	b := Body{"keyword": "ラッコ"}
	b.SetIf(false, "limit", 10)
	b.SetIf(true, "increaseKeyword", true)
	if _, ok := b["limit"]; ok {
		t.Error("unchanged flags must not reach the request body; the API's own default should apply")
	}
	if b["increaseKeyword"] != true {
		t.Error("changed flag missing from body")
	}
}

func TestEnumReportsAllowedValues(t *testing.T) {
	err := Enum("device", "tablet", []string{"desktop", "mobile"})
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "desktop, mobile") {
		t.Errorf("error should list the accepted values: %v", err)
	}
	if err := Enum("device", "", []string{"desktop", "mobile"}); err != nil {
		t.Errorf("an unset flag is not an invalid value: %v", err)
	}
}

func TestSplitList(t *testing.T) {
	got := SplitList([]string{"a, b", "", "c"})
	if strings.Join(got, "|") != "a|b|c" {
		t.Errorf("got %v", got)
	}
}

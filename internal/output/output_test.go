package output

import (
	"bytes"
	"strings"
	"testing"
)

const sample = `{
  "result": true,
  "meta": {"consumedCredit": 1.5},
  "data": {
    "query": {"keyword": "ラッコ"},
    "summary": {"totalCount": 864, "returnedCount": 2},
    "items": [
      {"keyword": "ラッコ", "metrics": {"searchVolume": 90500, "seoDifficulty": 35, "cpc": 0}, "engines": ["google", "bing"]},
      {"keyword": "らっこ", "metrics": {"searchVolume": 90500, "seoDifficulty": null, "cpc": 0.5}, "engines": ["google"]}
    ]
  },
  "errors": []
}`

func render(t *testing.T, opts Options) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Render(&buf, []byte(sample), opts); err != nil {
		t.Fatalf("Render: %v", err)
	}
	return buf.String()
}

func TestJSONIsThePayloadVerbatim(t *testing.T) {
	out := render(t, Options{Format: "json"})
	// Nothing may be dropped: agents parse this, and a Go struct in the middle
	// would silently lose any field the API adds.
	for _, want := range []string{`"consumedCredit"`, `"totalCount"`, `"engines"`, `"seoDifficulty": null`} {
		if !strings.Contains(out, want) {
			t.Errorf("json output lost %s:\n%s", want, out)
		}
	}
}

func TestJSONLEmitsOneItemPerLine(t *testing.T) {
	out := render(t, Options{Format: "jsonl", ItemsPath: "data.items"})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got %d:\n%s", len(lines), out)
	}
	// Key order comes from the response, not from Go's map iteration.
	if !strings.HasPrefix(lines[0], `{"keyword":"ラッコ","metrics":{"searchVolume":90500,`) {
		t.Errorf("field order not preserved: %s", lines[0])
	}
}

func TestCSVColumnsAreFlattenedPaths(t *testing.T) {
	out := render(t, Options{Format: "csv", ItemsPath: "data.items"})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	want := "keyword,metrics.searchVolume,metrics.seoDifficulty,metrics.cpc,engines"
	if lines[0] != want {
		t.Errorf("header\n got %s\nwant %s", lines[0], want)
	}
	if lines[1] != "ラッコ,90500,35,0,google|bing" {
		t.Errorf("row = %s", lines[1])
	}
	// A JSON null is an empty cell, not the string "null" or a zero.
	if !strings.Contains(lines[2], "らっこ,90500,,0.5,google") {
		t.Errorf("null handling: %s", lines[2])
	}
}

func TestFieldsSelectsColumns(t *testing.T) {
	out := render(t, Options{Format: "csv", ItemsPath: "data.items", Fields: []string{"keyword", "metrics.cpc"}})
	if !strings.HasPrefix(out, "keyword,metrics.cpc\n") {
		t.Errorf("--fields ignored:\n%s", out)
	}
}

func TestTablePrintsCaptionAndColumns(t *testing.T) {
	out := render(t, Options{
		Format:    "table",
		ItemsPath: "data.items",
		Caption:   []string{"data.query.keyword", "data.summary.totalCount"},
		Columns:   []string{"keyword", "metrics.searchVolume"},
	})
	if !strings.Contains(out, "keyword=ラッコ  totalCount=864") {
		t.Errorf("caption missing:\n%s", out)
	}
	if strings.Contains(out, "metrics.cpc") {
		t.Errorf("table should show only the requested columns:\n%s", out)
	}
}

func TestTableDropsColumnsNoRecordHas(t *testing.T) {
	// Commands share default column lists across responses whose optional
	// fields differ; an absent column must not become an empty one.
	out := render(t, Options{
		Format:    "table",
		ItemsPath: "data.items",
		Columns:   []string{"keyword", "metrics.relevance"},
	})
	if strings.Contains(out, "relevance") {
		t.Errorf("absent column should be dropped:\n%s", out)
	}
}

func TestTableRendersObjectResponsesAsKeyValue(t *testing.T) {
	status := `{"result":true,"data":{"isCompleted":false,"statuses":{"searchVolume":"processing"}},"errors":[]}`
	var buf bytes.Buffer
	if err := Render(&buf, []byte(status), Options{Format: "table"}); err != nil {
		t.Fatalf("Render: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"isCompleted", "false", "statuses.searchVolume", "processing"} {
		if !strings.Contains(out, want) {
			t.Errorf("status table lost %q:\n%s", want, out)
		}
	}
}

func TestTableTruncatesUnlessWide(t *testing.T) {
	long := `{"data":{"items":[{"title":"` + strings.Repeat("あ", 80) + `"}]}}`
	var narrow, wide bytes.Buffer
	_ = Render(&narrow, []byte(long), Options{Format: "table", ItemsPath: "data.items"})
	_ = Render(&wide, []byte(long), Options{Format: "table", ItemsPath: "data.items", Wide: true})
	if !strings.Contains(narrow.String(), "…") {
		t.Error("long values should be truncated by default")
	}
	if strings.Contains(wide.String(), "…") {
		t.Error("--wide should disable truncation")
	}
}

func TestDecodePreservesKeyOrderAndNumbers(t *testing.T) {
	v, err := Decode([]byte(`{"b":1,"a":2,"big":12345678901234567890,"frac":0.10}`))
	if err != nil {
		t.Fatal(err)
	}
	obj, ok := v.(*Object)
	if !ok {
		t.Fatalf("want *Object, got %T", v)
	}
	if strings.Join(obj.Keys, ",") != "b,a,big,frac" {
		t.Errorf("key order = %v", obj.Keys)
	}
	flat := Flatten(obj)
	// Large integers and trailing zeros survive because numbers are never
	// routed through float64.
	if flat["big"] != "12345678901234567890" {
		t.Errorf("big = %q", flat["big"])
	}
	if flat["frac"] != "0.10" {
		t.Errorf("frac = %q", flat["frac"])
	}
}

func TestEmptyItemsSaysSo(t *testing.T) {
	empty := `{"result":true,"data":{"summary":{"totalCount":0},"items":[]},"errors":[]}`
	var buf bytes.Buffer
	if err := Render(&buf, []byte(empty), Options{Format: "table", ItemsPath: "data.items"}); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(buf.String(), "no results") {
		t.Errorf("expected an explicit empty notice:\n%s", buf.String())
	}
}

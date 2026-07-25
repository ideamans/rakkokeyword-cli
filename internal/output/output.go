// Package output renders an API response as a table, JSON, JSONL or CSV.
//
// The rendering is generic on purpose. Every rakkokeyword endpoint answers with
// the same envelope and a list of records under data, so instead of a Go struct
// per endpoint — which would silently drop any field the API adds — the
// commands hand over the raw bytes plus the path to the list. JSON output is
// then the API's own bytes, and CSV columns are the response's own fields in
// the response's own order.
package output

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/olekukonko/tablewriter"
)

// Formats are the accepted --format values.
var Formats = []string{"table", "json", "jsonl", "csv"}

// ValidFormat reports whether name is a supported format.
func ValidFormat(name string) bool {
	for _, f := range Formats {
		if f == name {
			return true
		}
	}
	return false
}

// Options describes how to render one response.
type Options struct {
	// Format is one of Formats.
	Format string

	// ItemsPath is the dotted path to the array of records, e.g. "data.items".
	// When it is empty or absent from the response, the object at DataPath is
	// rendered as a single record instead.
	ItemsPath string

	// DataPath is the fallback object for responses that carry no array
	// (registration and status calls). Defaults to "data".
	DataPath string

	// Columns are the default table columns, as dotted paths into a record.
	// Empty means "the first MaxAutoColumns fields of the first record".
	Columns []string

	// Fields overrides Columns and applies to CSV as well (--fields).
	Fields []string

	// Caption paths are envelope values printed above the table, in order.
	Caption []string

	// Wide disables value truncation in table output.
	Wide bool
}

// MaxAutoColumns caps how many columns a table shows when the command has no
// preferred column list. Wide tables wrap in a terminal and become unreadable;
// the full record is always one `-f json` away.
const MaxAutoColumns = 8

// maxCellWidth is the truncation point for table cells (in runes).
const maxCellWidth = 44

// Render writes the response in the requested format.
func Render(w io.Writer, raw []byte, opts Options) error {
	if opts.DataPath == "" {
		opts.DataPath = "data"
	}

	if opts.Format == "json" {
		var buf bytes.Buffer
		if err := json.Indent(&buf, raw, "", "  "); err != nil {
			// Not valid JSON: print it as it came rather than hiding it.
			_, err := w.Write(raw)
			return err
		}
		buf.WriteByte('\n')
		_, err := w.Write(buf.Bytes())
		return err
	}

	root, err := Decode(raw)
	if err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	records, isList := recordsOf(root, opts)

	switch opts.Format {
	case "jsonl":
		return renderJSONL(w, records)
	case "csv":
		return renderCSV(w, records, opts)
	case "table":
		return renderTable(w, root, records, isList, opts)
	}
	return fmt.Errorf("unknown format %q (want %s)", opts.Format, strings.Join(Formats, ", "))
}

// recordsOf returns the records to render and whether they came from a list.
func recordsOf(root any, opts Options) ([]any, bool) {
	if opts.ItemsPath != "" {
		if v, ok := Lookup(root, opts.ItemsPath); ok {
			if arr, ok := v.([]any); ok {
				return arr, true
			}
		}
	}
	if v, ok := Lookup(root, opts.DataPath); ok {
		return []any{v}, false
	}
	return []any{root}, false
}

func renderJSONL(w io.Writer, records []any) error {
	enc := json.NewEncoder(w)
	for _, r := range records {
		if err := enc.Encode(r); err != nil {
			return err
		}
	}
	return nil
}

func renderCSV(w io.Writer, records []any, opts Options) error {
	columns := opts.Fields
	if len(columns) == 0 {
		columns = unionColumns(records)
	}
	cw := csv.NewWriter(w)
	if err := cw.Write(columns); err != nil {
		return err
	}
	for _, r := range records {
		flat := Flatten(r)
		row := make([]string, len(columns))
		for i, c := range columns {
			row[i] = flat[c]
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func renderTable(w io.Writer, root any, records []any, isList bool, opts Options) error {
	if caption := renderCaption(root, opts.Caption); caption != "" {
		fmt.Fprintln(w, caption)
	}

	if len(records) == 0 {
		fmt.Fprintln(w, "(no results)")
		return nil
	}

	// A single object — a status or registration response — reads better as
	// key/value pairs than as a one-row table with a dozen columns.
	if !isList && len(records) == 1 {
		flat := Flatten(records[0])
		keys := orderedKeys(records[0])
		t := tablewriter.NewWriter(w)
		t.SetHeader([]string{"field", "value"})
		t.SetBorder(false)
		t.SetColumnSeparator("  ")
		t.SetAutoWrapText(false)
		t.SetAutoFormatHeaders(false)
		for _, k := range keys {
			t.Append([]string{k, cell(flat[k], opts.Wide)})
		}
		t.Render()
		return nil
	}

	columns := opts.Fields
	if len(columns) == 0 {
		columns = opts.Columns
	}
	if len(columns) == 0 {
		columns = autoColumns(records)
	} else {
		columns = presentColumns(columns, records)
	}

	t := tablewriter.NewWriter(w)
	t.SetHeader(columns)
	t.SetBorder(false)
	t.SetColumnSeparator("  ")
	t.SetHeaderLine(true)
	t.SetAutoWrapText(false)
	t.SetAutoFormatHeaders(false)
	for _, r := range records {
		flat := Flatten(r)
		row := make([]string, len(columns))
		for i, c := range columns {
			row[i] = cell(flat[c], opts.Wide)
		}
		t.Append(row)
	}
	t.Render()
	return nil
}

// renderCaption builds the one-line summary above a table: what was asked for,
// how much came back and what it cost.
func renderCaption(root any, paths []string) string {
	var parts []string
	for _, p := range paths {
		v, ok := Lookup(root, p)
		if !ok || v == nil {
			continue
		}
		label := p
		if i := strings.LastIndex(p, "."); i >= 0 {
			label = p[i+1:]
		}
		parts = append(parts, fmt.Sprintf("%s=%s", label, scalar(v)))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "  ")
}

// presentColumns drops requested columns that no record has, so a shared
// default column list can cover responses whose optional fields are absent.
func presentColumns(columns []string, records []any) []string {
	seen := map[string]bool{}
	for _, r := range records {
		for k := range Flatten(r) {
			seen[k] = true
		}
	}
	var out []string
	for _, c := range columns {
		if seen[c] {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return autoColumns(records)
	}
	return out
}

// autoColumns takes the first MaxAutoColumns fields of the first record.
func autoColumns(records []any) []string {
	if len(records) == 0 {
		return nil
	}
	keys := orderedKeys(records[0])
	if len(keys) > MaxAutoColumns {
		keys = keys[:MaxAutoColumns]
	}
	return keys
}

// unionColumns is every field any record has, first-seen order preserved.
func unionColumns(records []any) []string {
	var out []string
	seen := map[string]bool{}
	for _, r := range records {
		for _, k := range orderedKeys(r) {
			if !seen[k] {
				seen[k] = true
				out = append(out, k)
			}
		}
	}
	return out
}

// orderedKeys returns a record's flattened paths in response order.
func orderedKeys(record any) []string {
	var keys []string
	walk(record, "", func(path string, _ any) { keys = append(keys, path) })
	return keys
}

// Flatten turns one record into dotted path -> formatted value.
func Flatten(record any) map[string]string {
	out := map[string]string{}
	walk(record, "", func(path string, v any) { out[path] = scalar(v) })
	return out
}

// walk visits every leaf of a record. Objects recurse into dotted paths; arrays
// of scalars collapse to one value; arrays of objects stay compact JSON,
// because expanding them would multiply rows and break the one-record-one-row
// contract that CSV consumers rely on.
func walk(v any, prefix string, visit func(path string, value any)) {
	switch t := v.(type) {
	case *Object:
		for _, k := range t.Keys {
			walk(t.Values[k], prefix+k+".", visit)
		}
	case []any:
		visit(strings.TrimSuffix(prefix, "."), t)
	default:
		visit(strings.TrimSuffix(prefix, "."), v)
	}
}

// scalar formats one value for a text cell.
func scalar(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case json.Number:
		return t.String()
	case []any:
		if len(t) == 0 {
			return ""
		}
		if allScalars(t) {
			parts := make([]string, len(t))
			for i, e := range t {
				parts[i] = scalar(e)
			}
			return strings.Join(parts, "|")
		}
		return compactJSON(t)
	case *Object:
		return compactJSON(t)
	}
	return fmt.Sprint(v)
}

func allScalars(arr []any) bool {
	for _, e := range arr {
		switch e.(type) {
		case *Object, []any:
			return false
		}
	}
	return true
}

func compactJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(data)
}

// cell truncates a value for table display unless --wide was given.
func cell(s string, wide bool) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if wide {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxCellWidth {
		return s
	}
	return string(runes[:maxCellWidth-1]) + "…"
}

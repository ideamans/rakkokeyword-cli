package rakko

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Body is a request body under construction. Commands only set the keys the
// user actually asked for, so the API's own defaults stay in charge of
// everything else.
type Body map[string]any

// Set assigns a top-level key.
func (b Body) Set(key string, value any) { b[key] = value }

// SetIf assigns key only when cond is true (typically flag.Changed).
func (b Body) SetIf(cond bool, key string, value any) {
	if cond {
		b[key] = value
	}
}

// ── enums ────────────────────────────────────────────────────────────────────

// Enum validates a flag value against the API's accepted values. The error
// lists them, because a wrong enum is otherwise a 400 from the server with a
// less specific message.
func Enum(flag, value string, allowed []string) error {
	if value == "" {
		return nil
	}
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return fmt.Errorf("invalid --%s %q: must be one of %s", flag, value, strings.Join(allowed, ", "))
}

// EnumList validates every element of a comma-separated list.
func EnumList(flag string, values []string, allowed []string) error {
	for _, v := range values {
		if err := Enum(flag, v, allowed); err != nil {
			return err
		}
	}
	return nil
}

// SplitList splits repeated and comma-separated flag values into one flat list,
// dropping empties.
func SplitList(raw []string) []string {
	var out []string
	for _, s := range raw {
		for _, part := range strings.Split(s, ",") {
			if p := strings.TrimSpace(part); p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}

// ── filters ──────────────────────────────────────────────────────────────────

// FilterKind is the JSON type a filter field expects.
type FilterKind int

const (
	// KindInt is a whole number, e.g. searchVolume.min.
	KindInt FilterKind = iota
	// KindFloat is a decimal number, e.g. cpc.max.
	KindFloat
	// KindStrings is a list of words; repeat the flag or comma-separate.
	KindStrings
	// KindInts is a list of whole numbers.
	KindInts
	// KindEnum is one of a fixed set of strings.
	KindEnum
)

// FilterField is one assignable path inside a command's filter object.
type FilterField struct {
	Path string // dotted path, e.g. "searchVolume.min"
	Kind FilterKind
	Enum []string // for KindEnum
}

// FilterSpec is the set of filter paths one endpoint accepts. Anything outside
// it is rejected locally: the API answers an unknown filter key with a generic
// 400, which is a slow and confusing way to learn about a typo.
type FilterSpec []FilterField

// Usage renders the accepted paths for a flag description.
func (s FilterSpec) Usage() string {
	if len(s) == 0 {
		return "this command takes no filters"
	}
	parts := make([]string, 0, len(s))
	for _, f := range s {
		switch f.Kind {
		case KindStrings:
			parts = append(parts, f.Path+"=<word,word>")
		case KindInts:
			parts = append(parts, f.Path+"=<n,n>")
		case KindEnum:
			parts = append(parts, f.Path+"="+strings.Join(f.Enum, "|"))
		case KindFloat:
			parts = append(parts, f.Path+"=<number>")
		default:
			parts = append(parts, f.Path+"=<int>")
		}
	}
	return strings.Join(parts, "\n")
}

func (s FilterSpec) lookup(path string) (FilterField, bool) {
	for _, f := range s {
		if f.Path == path {
			return f, true
		}
	}
	return FilterField{}, false
}

func (s FilterSpec) paths() []string {
	out := make([]string, 0, len(s))
	for _, f := range s {
		out = append(out, f.Path)
	}
	sort.Strings(out)
	return out
}

// ParseFilters turns `--filter path=value` assignments into the nested filter
// object the API expects. Repeating a list-valued path appends to it.
func (s FilterSpec) ParseFilters(assignments []string) (map[string]any, error) {
	if len(assignments) == 0 {
		return nil, nil
	}
	out := map[string]any{}
	for _, raw := range assignments {
		path, value, ok := strings.Cut(raw, "=")
		if !ok {
			return nil, fmt.Errorf("invalid --filter %q: expected path=value, e.g. searchVolume.min=100", raw)
		}
		path = strings.TrimSpace(path)
		field, known := s.lookup(path)
		if !known {
			return nil, fmt.Errorf("unknown filter path %q for this command; accepted: %s",
				path, strings.Join(s.paths(), ", "))
		}
		parsed, err := parseFilterValue(field, value)
		if err != nil {
			return nil, err
		}
		if err := assign(out, strings.Split(path, "."), parsed, field.Kind); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func parseFilterValue(f FilterField, value string) (any, error) {
	value = strings.TrimSpace(value)
	switch f.Kind {
	case KindInt:
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("filter %s: %q is not a whole number", f.Path, value)
		}
		return n, nil
	case KindFloat:
		n, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, fmt.Errorf("filter %s: %q is not a number", f.Path, value)
		}
		return n, nil
	case KindStrings:
		list := SplitList([]string{value})
		if len(list) == 0 {
			return nil, fmt.Errorf("filter %s: no values given", f.Path)
		}
		return list, nil
	case KindInts:
		var list []int64
		for _, part := range SplitList([]string{value}) {
			n, err := strconv.ParseInt(part, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("filter %s: %q is not a whole number", f.Path, part)
			}
			list = append(list, n)
		}
		if len(list) == 0 {
			return nil, fmt.Errorf("filter %s: no values given", f.Path)
		}
		return list, nil
	case KindEnum:
		for _, a := range f.Enum {
			if value == a {
				return value, nil
			}
		}
		return nil, fmt.Errorf("filter %s: %q must be one of %s", f.Path, value, strings.Join(f.Enum, ", "))
	}
	return value, nil
}

// assign writes value at the dotted path inside obj, appending when the path is
// list-valued and already present.
func assign(obj map[string]any, path []string, value any, kind FilterKind) error {
	key := path[0]
	if len(path) == 1 {
		if existing, ok := obj[key]; ok {
			switch kind {
			case KindStrings:
				obj[key] = append(existing.([]string), value.([]string)...)
				return nil
			case KindInts:
				obj[key] = append(existing.([]int64), value.([]int64)...)
				return nil
			}
		}
		obj[key] = value
		return nil
	}
	child, ok := obj[key].(map[string]any)
	if !ok {
		child = map[string]any{}
		obj[key] = child
	}
	return assign(child, path[1:], value, kind)
}

// MergeFilterJSON overlays a raw JSON filter object (the --filter-json escape
// hatch) onto filters built from --filter. Raw JSON wins on conflicts, since it
// is the more explicit of the two.
func MergeFilterJSON(filters map[string]any, raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return filters, nil
	}
	var extra map[string]any
	if err := json.Unmarshal([]byte(raw), &extra); err != nil {
		return nil, fmt.Errorf("--filter-json is not a JSON object: %w", err)
	}
	if filters == nil {
		return extra, nil
	}
	for k, v := range extra {
		filters[k] = v
	}
	return filters, nil
}

// ── shared filter fragments ──────────────────────────────────────────────────

// The API reuses the same filter shapes across endpoints, so the specs below
// are assembled from these fragments rather than restated per command.

func rangeInt(name string) FilterSpec {
	return FilterSpec{{Path: name + ".min", Kind: KindInt}, {Path: name + ".max", Kind: KindInt}}
}

func rangeFloat(name string) FilterSpec {
	return FilterSpec{{Path: name + ".min", Kind: KindFloat}, {Path: name + ".max", Kind: KindFloat}}
}

func words(name string) FilterSpec {
	return FilterSpec{
		{Path: name + ".includes", Kind: KindStrings},
		{Path: name + ".notIncludes", Kind: KindStrings},
	}
}

var firstSeenValues = []string{
	"last_7_days", "last_30_days", "last_90_days", "within_6_months", "within_1_year", "over_1_year",
}

func firstSeen() FilterSpec {
	return FilterSpec{{Path: "firstSeenRange.include", Kind: KindEnum, Enum: firstSeenValues}}
}

func concat(specs ...FilterSpec) FilterSpec {
	var out FilterSpec
	for _, s := range specs {
		out = append(out, s...)
	}
	return out
}

// seoMetrics is the block shared by every keyword-shaped endpoint.
func seoMetrics() FilterSpec {
	return concat(rangeInt("seoDifficulty"), rangeInt("searchVolume"), rangeFloat("cpc"), rangeInt("competition"))
}

// Filter specs, one per endpoint that accepts a filter object.
var (
	// SuggestFilters is the filter for POST /v1/suggest-keywords.
	SuggestFilters = concat(
		FilterSpec{{Path: "suggestClass", Kind: KindInts}},
		words("keyword"), seoMetrics(), firstSeen(),
	)

	// RelatedFilters is the filter for POST /v1/related-keywords.
	RelatedFilters = concat(words("keyword"), seoMetrics(), firstSeen())

	// RankingFilters is the filter for POST /v1/ranking-keywords.
	RankingFilters = concat(words("keyword"), seoMetrics(), rangeInt("relevance"))

	// SearchVolumeResultFilters is the filter for POST /v1/search-volume/{id}/results.
	SearchVolumeResultFilters = concat(words("keyword"), seoMetrics(), firstSeen())

	// SearchRankResultFilters is the filter for POST /v1/search-rank/{id}/results.
	SearchRankResultFilters = concat(words("keyword"), rangeInt("seoDifficulty"), rangeInt("searchVolume"))

	// InfluxKeywordsFilters is the filter for POST /v1/influx-keywords.
	InfluxKeywordsFilters = concat(
		words("keyword"), rangeInt("seoDifficulty"), rangeInt("rank"), rangeInt("searchVolume"),
		rangeFloat("cpc"), rangeInt("competition"), rangeInt("etv"),
	)

	// InfluxPagesFilters is the filter for POST /v1/influx-pages.
	InfluxPagesFilters = concat(
		rangeInt("totalEtv"), rangeInt("keywordCount"), rangeInt("totalTrafficValue"),
		words("title"), words("url"), words("topKeyword"), rangeInt("topSeoDifficulty"),
	)

	// ContentSearchFilters is the filter for POST /v1/content-search.
	ContentSearchFilters = concat(
		rangeInt("estimatedTraffic"), rangeInt("rankingKeywordCount"), rangeInt("trafficValue"),
		words("title"), words("url"), words("topKeyword"), words("description"), rangeInt("seoDifficulty"),
	)

	// SiteSearchFilters is the filter for POST /v1/site-search.
	SiteSearchFilters = concat(
		words("keyword"), words("domain"),
		FilterSpec{{Path: "domain.matchType", Kind: KindEnum, Enum: []string{"partialMatch", "prefixMatch", "suffixMatch"}}},
		rangeInt("totalEtv"), rangeInt("keywordCount"), rangeInt("pageCount"),
		rangeInt("totalTrafficValue"), rangeInt("relatedContentEtv"), rangeInt("contentRelevance"),
	)
)

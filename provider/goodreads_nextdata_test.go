package provider

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/prairie-server/prairie-plugin-metadata-ebook/metadata"
)

func TestParseGoodreadsNextData(t *testing.T) {
	book := map[string]any{
		"title":           "Project Hail Mary",
		"description":     "A lone astronaut.",
		"imageUrl":        "https://example.invalid/cover.jpg",
		"isbn":            "9780593135204",
		"publisher":       "Ballantine",
		"language":        "eng",
		"publicationDate": "2021-05-04",
		"legacyId":        float64(54493401),
		"numPages":        float64(476),
		"authors": []any{
			map[string]any{"name": "Andy Weir"},
			map[string]any{"name": ""},
			"skip",
		},
		"genres": []any{
			"Science Fiction",
			map[string]any{"name": "Space"},
			map[string]any{"name": ""},
			42,
		},
	}
	payload, err := json.Marshal(map[string]any{
		"props": map[string]any{
			"pageProps": map[string]any{
				"apolloState": map[string]any{
					"Book:1": book,
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	html := `<html><script id="__NEXT_DATA__" type="application/json">` + string(payload) + `</script></html>`

	match := parseGoodreadsNextData(html)
	if match == nil {
		t.Fatal("parseGoodreadsNextData returned nil")
	}
	if match.Title != "Project Hail Mary" || match.ProviderID != "54493401" {
		t.Fatalf("match = %#v", match)
	}
	if match.ISBN != "9780593135204" || match.PublishYear != 2021 || match.PageCount != 476 {
		t.Fatalf("detail fields = %#v", match)
	}
	if len(match.Authors) != 1 || match.Authors[0] != "Andy Weir" {
		t.Fatalf("Authors = %#v", match.Authors)
	}
	if len(match.Genres) != 2 {
		t.Fatalf("Genres = %#v", match.Genres)
	}
}

func TestParseGoodreadsNextDataFallbacks(t *testing.T) {
	if parseGoodreadsNextData(`<html></html>`) != nil {
		t.Fatal("expected nil without next data")
	}
	if parseGoodreadsNextData(`<script id="__NEXT_DATA__">{not-json}</script>`) != nil {
		t.Fatal("expected nil for invalid json")
	}
	if parseGoodreadsNextData(`<script id="__NEXT_DATA__">{"title":"Only Title"}</script>`) != nil {
		t.Fatal("expected nil for title-only object")
	}

	nested := `{"items":[{"child":{"title":"Nested","isbn":"9780000000000","id":"abc123"}}]}`
	match := parseGoodreadsNextData(`<script id="__NEXT_DATA__">` + nested + `</script>`)
	if match == nil || match.ProviderID != "abc123" || match.Title != "Nested" {
		t.Fatalf("nested match = %#v", match)
	}
}

func TestTraverseGoodreadsNextDataDepthLimit(t *testing.T) {
	var root any = "leaf"
	for i := 0; i < maxGoodreadsTraverseDepth+3; i++ {
		root = []any{root}
	}
	var out []metadata.Match
	traverseGoodreadsNextData(root, &out, 0)
	if len(out) != 0 {
		t.Fatalf("expected no matches past depth, got %d", len(out))
	}
}

func TestGoodreadsNextDataBookToMatchAndHelpers(t *testing.T) {
	if goodreadsNextDataBookToMatch(map[string]any{}) != nil {
		t.Fatal("expected nil without title")
	}
	if goodreadsNextDataBookToMatch(map[string]any{"title": "X"}) != nil {
		t.Fatal("expected nil without book fields")
	}

	match := goodreadsNextDataBookToMatch(map[string]any{
		"title":       "  Book  ",
		"description": "d",
		"id":          "99",
	})
	if match == nil || match.Title != "Book" || match.ProviderID != "99" {
		t.Fatalf("match = %#v", match)
	}

	if grStringField(map[string]any{"x": 1}, "x") != "" {
		t.Fatal("non-string field should be empty")
	}
	if grStringField(map[string]any{"x": "  hi "}, "x") != "hi" {
		t.Fatal("string field trim failed")
	}
	if grStringField(map[string]any{}, "missing") != "" {
		t.Fatal("missing field")
	}

	names := grExtractNames(json.RawMessage(`[{"name":"A"},{"name":""},{"name":"B"}]`))
	if len(names) != 2 || names[0] != "A" || names[1] != "B" {
		t.Fatalf("array names = %#v", names)
	}
	names = grExtractNames(json.RawMessage(`{"name":"Solo"}`))
	if len(names) != 1 || names[0] != "Solo" {
		t.Fatalf("single name = %#v", names)
	}
	if names := grExtractNames(json.RawMessage(`true`)); len(names) != 0 {
		t.Fatalf("expected empty for invalid raw, got %#v", names)
	}
	if names := grExtractNames(json.RawMessage(`"nope"`)); len(names) != 1 || names[0] != "nope" {
		t.Fatalf("string raw = %#v", names)
	}
	if grExtractNames(json.RawMessage(`{"name":""}`)) != nil {
		t.Fatal("expected nil for empty single name")
	}

	if goodreadsIDFromURL("") != "" {
		t.Fatal("empty url")
	}
	if got := goodreadsIDFromURL("https://www.goodreads.com/book/show/12345-Title"); got != "12345" {
		t.Fatalf("id = %q", got)
	}
	if goodreadsIDFromURL("https://example.invalid/nope") != "" {
		t.Fatal("expected empty for non-match")
	}

	if grJSONLDString(nil, "name") != "" {
		t.Fatal("nil doc")
	}
	if grJSONLDString(map[string]json.RawMessage{}, "name") != "" {
		t.Fatal("missing key")
	}
	if grJSONLDString(map[string]json.RawMessage{"name": json.RawMessage(`42`)}, "name") != "" {
		t.Fatal("non-string json")
	}
	if grJSONLDString(map[string]json.RawMessage{"name": json.RawMessage(`"  pub  "`)}, "name") != "pub" {
		t.Fatal("string trim")
	}

	if grJSONLDPublisher(nil) != "" {
		t.Fatal("nil publisher")
	}
	if grJSONLDPublisher(json.RawMessage(`{"name":"House"}`)) != "House" {
		t.Fatal("object publisher")
	}
	if grJSONLDPublisher(json.RawMessage(`"Direct"`)) != "Direct" {
		t.Fatal("string publisher")
	}
	if grJSONLDPublisher(json.RawMessage(`true`)) != "" {
		t.Fatal("invalid publisher")
	}

	page := parseGoodreadsBookPage([]byte(`<script id="__NEXT_DATA__">{"title":"Via Next","genres":["G"]}</script>`))
	if page == nil || page.Title != "Via Next" {
		t.Fatalf("book page fallback = %#v", page)
	}

	var b strings.Builder
	for i := 0; i < grMaxResults+2; i++ {
		b.WriteString(`href="/book/show/1-x"><span itemprop="name">T</span>`)
	}
	search := parseGoodreadsSearchPage([]byte(b.String()))
	if len(search) != grMaxResults {
		t.Fatalf("search capped = %d, want %d", len(search), grMaxResults)
	}
	if parseGoodreadsSearchPage([]byte(`no rows`)) != nil {
		t.Fatal("expected nil search")
	}
}

func TestGoodreadsSearchEmptyQuery(t *testing.T) {
	client := NewGoodreadsClientAt("http://example.invalid", "ua")
	matches, err := client.Search(context.Background(), metadata.SearchQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if matches != nil {
		t.Fatalf("want nil, got %#v", matches)
	}
}

package provider

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prairie-server/prairie-plugin-metadata-ebook/metadata"
)

type edgeErrReadCloser struct{}

func (edgeErrReadCloser) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (edgeErrReadCloser) Close() error             { return nil }

func TestHTMLTextAndAAHelpers(t *testing.T) {
	got := htmlText(`<b>Hi&nbsp;&amp;&#65;</b>`)
	if got != "Hi &A" {
		t.Fatalf("htmlText = %q", got)
	}
	if aaResolveURL("", "https://x") != "" {
		t.Fatal("empty")
	}
	if aaResolveURL("https://a/b", "https://x") != "https://a/b" {
		t.Fatal("absolute")
	}
	if aaResolveURL("//cdn/x", "https://x") != "https://cdn/x" {
		t.Fatal("protocol-relative https")
	}
	if aaResolveURL("//cdn/x", "http://x") != "http://cdn/x" {
		t.Fatal("protocol-relative http")
	}
	if aaResolveURL("/p", "https://x/") != "https://x/p" {
		t.Fatal("root-relative")
	}
	if aaResolveURL("rel", "https://x") != "rel" {
		t.Fatal("relative passthrough")
	}
	if aaStripText("") != "" {
		t.Fatal("empty strip")
	}
	got = aaStripText(`<i>A&nbsp;B&#66;</i>`)
	if got == "" {
		t.Fatal("strip empty")
	}
	if single("") != nil || single("  x  ")[0] != "x" {
		t.Fatal("single")
	}
	if NewProvider() == nil {
		t.Fatal("NewProvider")
	}
	p := NewProviderWithSources([]Source{nil, &fakeSource{id: ""}, &fakeSource{id: "ok"}})
	if len(p.sources) != 1 {
		t.Fatalf("sources = %d", len(p.sources))
	}
}

func TestGoogleBooksCoverAndToMatch(t *testing.T) {
	if googleBooksCoverURL(nil) != "" {
		t.Fatal("nil links")
	}
	u := googleBooksCoverURL(&googleBooksImageLinks{Thumbnail: "http://img/t"})
	if u != "https://img/t" {
		t.Fatalf("cover = %q", u)
	}
	m := googleBooksVolume{
		ID: "1",
		VolumeInfo: googleBooksVolumeInfo{
			Title: "T",
			IndustryIdentifiers: []googleBooksIdentifier{
				{Type: "ISBN_10", Identifier: "0593135202"},
			},
			ImageLinks: &googleBooksImageLinks{Small: "http://img/s"},
		},
	}.toMatch()
	if m.ISBN != "0593135202" || m.CoverURL != "https://img/s" {
		t.Fatalf("%#v", m)
	}
	m = googleBooksVolume{
		ID: "2",
		VolumeInfo: googleBooksVolumeInfo{
			IndustryIdentifiers: []googleBooksIdentifier{{Type: "ISBN_13", Identifier: "9780593135204"}},
		},
	}.toMatch()
	if m.ISBN != "9780593135204" {
		t.Fatalf("%#v", m)
	}
}

func TestCatalogEmptySearchesAndBadIDs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newCatalogHTTPClient(srv.URL, "ua")
	c.client = srv.Client()
	_, status, err := c.get(context.Background(), srv.URL+"/missing")
	if status != http.StatusNotFound || err == nil {
		t.Fatalf("status=%d err=%v", status, err)
	}

	empty := metadata.SearchQuery{}
	g := NewGutenbergClientAt(srv.URL, "ua")
	g.http.client = srv.Client()
	if matches, err := g.Search(context.Background(), empty); err != nil || matches != nil {
		t.Fatalf("gutenberg empty: %v %#v", err, matches)
	}
	if m, err := g.Fetch(context.Background(), "abc"); err != nil || m != nil {
		t.Fatalf("gutenberg bad id: %v %#v", err, m)
	}

	bb := NewBookBrainzClientAt(srv.URL, "ua")
	bb.http.client = srv.Client()
	if matches, err := bb.Search(context.Background(), empty); err != nil || matches != nil {
		t.Fatalf("bookbrainz empty: %v %#v", err, matches)
	}
	if m, err := bb.Fetch(context.Background(), "not-uuid"); err != nil || m != nil {
		t.Fatalf("bookbrainz bad id: %v %#v", err, m)
	}

	ff := NewFantasticFictionClientAt(srv.URL, "ua")
	ff.http.client = srv.Client()
	if matches, err := ff.Search(context.Background(), empty); err != nil || matches != nil {
		t.Fatalf("ff empty: %v %#v", err, matches)
	}

	isfdb := NewISFDBClientAt(srv.URL, "ua")
	isfdb.http.client = srv.Client()
	if matches, err := isfdb.Search(context.Background(), empty); err != nil || matches != nil {
		t.Fatalf("isfdb empty: %v %#v", err, matches)
	}

	lt := NewLibraryThingClientAt(srv.URL, "ua")
	lt.http.client = srv.Client()
	if matches, err := lt.Search(context.Background(), empty); err != nil || matches != nil {
		t.Fatalf("lt empty: %v %#v", err, matches)
	}

	ia := NewInternetArchiveClientAt(srv.URL, "ua")
	ia.http.client = srv.Client()
	if matches, err := ia.Search(context.Background(), empty); err != nil || matches != nil {
		t.Fatalf("ia empty: %v %#v", err, matches)
	}

	wc := NewWorldCatClientAt(srv.URL, "ua")
	wc.http.client = srv.Client()
	if matches, err := wc.Search(context.Background(), empty); err != nil || matches != nil {
		t.Fatalf("wc empty: %v %#v", err, matches)
	}

	db := NewDoubanClientAt(srv.URL, "ua")
	db.http.client = srv.Client()
	if matches, err := db.Search(context.Background(), empty); err != nil || matches != nil {
		t.Fatalf("douban empty: %v %#v", err, matches)
	}
}

func TestHTTPHelperEdgeErrors(t *testing.T) {
	if redactURL(nil) != "" {
		t.Fatal("nil URL should redact to empty")
	}
	if got := redactRawURL("http://%zz"); got != "http://%zz" {
		t.Fatalf("redactRawURL invalid = %q", got)
	}
	if _, err := httpGetBytes(context.Background(), http.DefaultClient, "http://[::1", "ua"); err == nil {
		t.Fatal("expected invalid URL error")
	}

	req, err := http.NewRequest(http.MethodGet, "https://example.test/md5/secret?api_key=secret", nil)
	if err != nil {
		t.Fatal(err)
	}
	requestErrClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("boom")
	})}
	if _, _, err := httpDoBytes(context.Background(), requestErrClient, req); err == nil {
		t.Fatal("expected httpDoBytes request error")
	}
	if _, err := httpGetBytes(context.Background(), requestErrClient, req.URL.String(), "ua"); err == nil {
		t.Fatal("expected httpGetBytes request error")
	}

	readErrClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: edgeErrReadCloser{}, Header: make(http.Header), Request: req}, nil
	})}
	if _, _, err := httpDoBytes(context.Background(), readErrClient, req); err == nil {
		t.Fatal("expected httpDoBytes read error")
	}
	if _, err := httpGetBytes(context.Background(), readErrClient, req.URL.String(), "ua"); err == nil {
		t.Fatal("expected httpGetBytes read error")
	}

	tooLarge := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(io.LimitReader(zeroReader{}, maxResponseBytes+1)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})}
	if _, _, err := httpDoBytes(context.Background(), tooLarge, req); err == nil {
		t.Fatal("expected httpDoBytes too-large error")
	}
	if _, err := httpGetBytes(context.Background(), tooLarge, req.URL.String(), "ua"); err == nil {
		t.Fatal("expected httpGetBytes too-large error")
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

func TestOpenLibraryMappingAndErrorEdges(t *testing.T) {
	if results, err := NewOpenLibraryClientAt("https://example.test", "", "ua").Search(context.Background(), metadata.SearchQuery{}); err != nil || results != nil {
		t.Fatalf("empty search = %#v err=%v", results, err)
	}
	if match, err := NewOpenLibraryClientAt("https://example.test", "", "ua").Fetch(context.Background(), ""); err != nil || match != nil {
		t.Fatalf("empty fetch = %#v err=%v", match, err)
	}
	if got := openLibraryISBN(nil, []string{"0593135202"}); got != "0593135202" {
		t.Fatalf("isbn10 fallback = %q", got)
	}

	edition := openLibraryEdition{
		Key:         "/books/OL1M",
		Title:       "Mapped",
		Description: map[string]any{"value": "map description"},
		ISBN10:      []string{"0593135202"},
	}
	if got := edition.toMatch("").Description; got != "map description" {
		t.Fatalf("description = %q", got)
	}
	doc := openLibrarySearchDoc{Key: "/books/OL123M", Title: "Doc"}
	if got := doc.toMatch("").ProviderID; got != "OL123M" {
		t.Fatalf("provider id from key = %q", got)
	}
	if got := (openLibrarySearchDoc{Key: "/works/OL1W", Title: "No Edition"}).toMatch("").ProviderID; got != "" {
		t.Fatalf("unexpected provider id = %q", got)
	}

	notFound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer notFound.Close()
	ol404 := NewOpenLibraryClientAt(notFound.URL, "", "ua")
	ol404.client = notFound.Client()
	if match, err := ol404.Fetch(context.Background(), "OL123M"); err != nil || match != nil {
		t.Fatalf("edition 404 = %#v err=%v", match, err)
	}
	if results, err := ol404.Search(context.Background(), metadata.SearchQuery{ProviderIDs: map[string]string{"isbn": "9780593135204"}}); err != nil || results != nil {
		t.Fatalf("isbn fallback 404 = %#v err=%v", results, err)
	}

	badJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not-json"))
	}))
	defer badJSON.Close()
	olBad := NewOpenLibraryClientAt(badJSON.URL, "", "ua")
	olBad.client = badJSON.Client()
	if _, err := olBad.Search(context.Background(), metadata.SearchQuery{Title: "x"}); err == nil {
		t.Fatal("expected openlibrary search decode error")
	}
	if _, err := olBad.Fetch(context.Background(), "OL123M"); err == nil {
		t.Fatal("expected openlibrary fetch decode error")
	}
	olInvalid := NewOpenLibraryClientAt("http://[::1", "", "ua")
	if _, err := olInvalid.Fetch(context.Background(), "OL123M"); err == nil {
		t.Fatal("expected openlibrary create request error")
	}
}

func TestHardcoverEdges(t *testing.T) {
	hc := NewHardcoverClientAt("https://example.test", "key", "ua")
	if results, err := hc.Search(context.Background(), metadata.SearchQuery{}); err != nil || results != nil {
		t.Fatalf("empty search = %#v err=%v", results, err)
	}
	if asciiDigits("") {
		t.Fatal("empty digits")
	}
	if match, err := hc.Fetch(context.Background(), "１２３"); err != nil || match != nil {
		t.Fatalf("unicode digits fetch = %#v err=%v", match, err)
	}
	if got := (hardcoverBook{ID: 7, Editions: []hardcoverEdition{{ISBN10: "0593135202"}}}).toMatch().ISBN; got != "0593135202" {
		t.Fatalf("hardcover ISBN10 fallback = %q", got)
	}

	errorSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":[{"message":"bad query"}]}`))
	}))
	defer errorSrv.Close()
	hcErr := NewHardcoverClientAt(errorSrv.URL, "key", "ua")
	hcErr.client = errorSrv.Client()
	if _, err := hcErr.Search(context.Background(), metadata.SearchQuery{Title: "x"}); err == nil {
		t.Fatal("expected graphql envelope error")
	}

	notFound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer notFound.Close()
	hc404 := NewHardcoverClientAt(notFound.URL, "key", "ua")
	hc404.client = notFound.Client()
	if body, err := hc404.graphql(context.Background(), "query", nil); err != nil || body != nil {
		t.Fatalf("graphql 404 = %q err=%v", body, err)
	}

	badJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not-json"))
	}))
	defer badJSON.Close()
	hcBad := NewHardcoverClientAt(badJSON.URL, "key", "ua")
	hcBad.client = badJSON.Client()
	if _, err := hcBad.Search(context.Background(), metadata.SearchQuery{Title: "x"}); err == nil {
		t.Fatal("expected hardcover search decode error")
	}
	if _, err := hcBad.Fetch(context.Background(), "1"); err == nil {
		t.Fatal("expected hardcover fetch decode error")
	}

	hcInvalid := NewHardcoverClientAt("http://[::1", "key", "ua")
	if _, err := hcInvalid.graphql(context.Background(), "query", nil); err == nil {
		t.Fatal("expected hardcover create request error")
	}
}

func TestAnnasArchiveParserEdges(t *testing.T) {
	detail := []byte(`<html><head><title>Title From Tag - Anna's Archive</title></head><body>
		<img src="//cdn.example.test/c.jpg" alt="cover">
		Author: <a>Unknown</a>
		Publisher: <a>Pub</a>
		Year: 2020
		ISBN-10: 0593135202
		Language: <a>English</a>
		Pages: nope
		<div class="description">Desc &amp; more</div>
		Extension: epub
	</body></html>`)
	match, ext := parseAnnasArchiveDetailPage(detail, "http://annas.test")
	if match == nil || match.Title != "Title From Tag" || ext != "epub" {
		t.Fatalf("detail = %#v ext=%q", match, ext)
	}
	if match.Language != "en" || len(match.Authors) != 0 || match.ISBN != "0593135202" {
		t.Fatalf("mapped detail = %#v", match)
	}
	if match.CoverURL != "http://cdn.example.test/c.jpg" {
		t.Fatalf("cover = %q", match.CoverURL)
	}
	if match, ext := parseAnnasArchiveDetailPage([]byte(`<html><title></title></html>`), "https://x"); match != nil || ext != "" {
		t.Fatalf("empty detail = %#v/%q", match, ext)
	}
	if got := parseAnnasArchiveSearchPage([]byte(`<p>no rows</p>`), "https://x"); got != nil {
		t.Fatalf("no rows = %#v", got)
	}
	search := []byte(`<table>
	<tr><td>no md5</td></tr>
	<tr><td><a href="/md5/a1b2c3d4e5f67890abcdef1234567890" class="js-vim-focus">Alt Title<</a></td><td>2021 [FR] 0593135202 <img src="/c.jpg"> epub</td></tr>
	<tr><td><a href="/md5/a1b2c3d4e5f67890abcdef1234567890"><span>Dupe</span></a></td><td>epub</td></tr>
	<tr><td><a href="/md5/b1b2c3d4e5f67890abcdef1234567890"><span>X</span></a></td><td>mp3</td></tr>
	</table>`)
	results := parseAnnasArchiveSearchPage(search, "https://annas.test")
	if len(results) != 1 {
		t.Fatalf("search results = %#v", results)
	}
	if results[0].Language != "fr" || results[0].CoverURL != "https://annas.test/c.jpg" {
		t.Fatalf("search mapped = %#v", results[0])
	}
	client := NewAnnasArchiveClientAt("http://[::1", "ua")
	if _, err := client.Fetch(context.Background(), "a1b2c3d4e5f67890abcdef1234567890"); err == nil {
		t.Fatal("expected annas create request error")
	}
}

func TestGoogleBooksISBNdbGoodreadsEdges(t *testing.T) {
	if results, err := NewGoogleBooksClientAt("https://example.test", "key", "ua").Search(context.Background(), metadata.SearchQuery{}); err != nil || results != nil {
		t.Fatalf("google empty search = %#v err=%v", results, err)
	}
	if googleBooksCoverURL(&googleBooksImageLinks{}) != "" {
		t.Fatal("empty google image links")
	}
	gbEmpty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	defer gbEmpty.Close()
	gb := NewGoogleBooksClientAt(gbEmpty.URL, "key", "ua")
	gb.client = gbEmpty.Client()
	if match, err := gb.Fetch(context.Background(), "9780593135204"); err != nil || match != nil {
		t.Fatalf("google isbn no matches = %#v err=%v", match, err)
	}
	gbBad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not-json"))
	}))
	defer gbBad.Close()
	gbBadClient := NewGoogleBooksClientAt(gbBad.URL, "key", "ua")
	gbBadClient.client = gbBad.Client()
	if _, err := gbBadClient.Search(context.Background(), metadata.SearchQuery{Title: "x"}); err == nil {
		t.Fatal("expected google search decode error")
	}
	if _, err := gbBadClient.Fetch(context.Background(), "zyTCAlFPjgYC"); err == nil {
		t.Fatal("expected google fetch decode error")
	}
	gbInvalid := NewGoogleBooksClientAt("http://[::1", "key", "ua")
	if _, err := gbInvalid.Fetch(context.Background(), "zyTCAlFPjgYC"); err == nil {
		t.Fatal("expected google create request error")
	}

	if results, err := NewISBNdbClientAt("https://example.test", "key", "ua").Search(context.Background(), metadata.SearchQuery{}); err != nil || results != nil {
		t.Fatalf("isbndb empty search = %#v err=%v", results, err)
	}
	isbn404 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer isbn404.Close()
	isbn404Client := NewISBNdbClientAt(isbn404.URL, "key", "ua")
	isbn404Client.client = isbn404.Client()
	if results, err := isbn404Client.Search(context.Background(), metadata.SearchQuery{Title: "x"}); err != nil || results != nil {
		t.Fatalf("isbndb search 404 = %#v err=%v", results, err)
	}
	if match, err := isbn404Client.Fetch(context.Background(), "9780593135204"); err != nil || match != nil {
		t.Fatalf("isbndb fetch 404 = %#v err=%v", match, err)
	}
	isbnBad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not-json"))
	}))
	defer isbnBad.Close()
	isbnBadClient := NewISBNdbClientAt(isbnBad.URL, "key", "ua")
	isbnBadClient.client = isbnBad.Client()
	if _, err := isbnBadClient.Search(context.Background(), metadata.SearchQuery{Title: "x"}); err == nil {
		t.Fatal("expected isbndb search decode error")
	}
	if _, err := isbnBadClient.Fetch(context.Background(), "9780593135204"); err == nil {
		t.Fatal("expected isbndb fetch decode error")
	}
	isbnNilBook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"book":null}`))
	}))
	defer isbnNilBook.Close()
	isbnNilClient := NewISBNdbClientAt(isbnNilBook.URL, "key", "ua")
	isbnNilClient.client = isbnNilBook.Client()
	if match, err := isbnNilClient.Fetch(context.Background(), "9780593135204"); err != nil || match != nil {
		t.Fatalf("isbndb nil book = %#v err=%v", match, err)
	}
	isbnInvalid := NewISBNdbClientAt("http://[::1", "key", "ua")
	if _, err := isbnInvalid.Search(context.Background(), metadata.SearchQuery{Title: "x"}); err == nil {
		t.Fatal("expected isbndb search create request error")
	}
	if _, err := isbnInvalid.Fetch(context.Background(), "9780593135204"); err == nil {
		t.Fatal("expected isbndb fetch create request error")
	}

	if results, err := NewGoodreadsClientAt("https://example.test", "ua").Search(context.Background(), metadata.SearchQuery{}); err != nil || results != nil {
		t.Fatalf("goodreads empty search = %#v err=%v", results, err)
	}
	if parseGoodreadsBookPage([]byte(`<html></html>`)) != nil {
		t.Fatal("empty goodreads page")
	}
	jsonldNoURL := []byte(`<script type="application/ld+json">{"@type":"Book","name":"No URL"}</script>`)
	grSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(jsonldNoURL)
	}))
	defer grSrv.Close()
	grClient := NewGoodreadsClientAt(grSrv.URL, "ua")
	grClient.client = grSrv.Client()
	match, err := grClient.Fetch(context.Background(), "123")
	if err != nil || match == nil || match.ProviderID != "123" {
		t.Fatalf("goodreads provider fallback = %#v err=%v", match, err)
	}
	if parseGoodreadsJSONLD(`<script type="application/ld+json">{bad</script><script type="application/ld+json">{"@type":"Movie"}</script>`) != nil {
		t.Fatal("invalid/non-book jsonld")
	}
	if goodreadsNextDataBookToMatch(map[string]any{"title": "Only Title"}) != nil {
		t.Fatal("next data title-only should be ignored")
	}
	var out []metadata.Match
	traverseGoodreadsNextData(map[string]any{"title": "Too Deep", "isbn": "9780593135204"}, &out, maxGoodreadsTraverseDepth+1)
	if len(out) != 0 {
		t.Fatalf("deep traverse = %#v", out)
	}
}

func TestCatalogSourceAdditionalEdges(t *testing.T) {
	if _, _, err := newCatalogHTTPClient("https://example.test", "ua").get(context.Background(), "http://[::1"); err == nil {
		t.Fatal("catalog get invalid URL")
	}
	if got := (gutenbergBook{ID: 1, Formats: map[string]string{"image/png": "png"}}).toMatch().CoverURL; got != "png" {
		t.Fatalf("gutenberg png cover = %q", got)
	}
	badJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not-json"))
	}))
	defer badJSON.Close()
	g := NewGutenbergClientAt(badJSON.URL, "ua")
	g.http.client = badJSON.Client()
	if _, err := g.Search(context.Background(), metadata.SearchQuery{Title: "x"}); err == nil {
		t.Fatal("expected gutenberg search decode error")
	}
	if _, err := g.Fetch(context.Background(), "1"); err == nil {
		t.Fatal("expected gutenberg fetch decode error")
	}
	bb := NewBookBrainzClientAt(badJSON.URL, "ua")
	bb.http.client = badJSON.Client()
	if _, err := bb.Search(context.Background(), metadata.SearchQuery{Title: "x"}); err == nil {
		t.Fatal("expected bookbrainz search decode error")
	}
	if _, err := bb.Fetch(context.Background(), bbValidID); err == nil {
		t.Fatal("expected bookbrainz fetch decode error")
	}

	notFound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer notFound.Close()
	for name, run := range map[string]func() (*metadata.Match, error){
		"fantasticfiction empty": func() (*metadata.Match, error) {
			return NewFantasticFictionClientAt(notFound.URL, "ua").Fetch(context.Background(), "")
		},
		"isfdb invalid": func() (*metadata.Match, error) {
			return NewISFDBClientAt(notFound.URL, "ua").Fetch(context.Background(), "abc")
		},
		"internetarchive empty": func() (*metadata.Match, error) {
			return NewInternetArchiveClientAt(notFound.URL, "ua").Fetch(context.Background(), "")
		},
		"douban invalid": func() (*metadata.Match, error) {
			return NewDoubanClientAt(notFound.URL, "ua").Fetch(context.Background(), "abc")
		},
	} {
		match, err := run()
		if err != nil || match != nil {
			t.Fatalf("%s = %#v err=%v", name, match, err)
		}
	}
}

func TestFinalSmallCoverageEdges(t *testing.T) {
	p := NewProviderWithSources([]Source{&fakeSource{id: "openlibrary"}})
	if match, err := p.Fetch(context.Background(), metadata.SearchQuery{}); err != nil || match != nil {
		t.Fatalf("provider fetch without isbn = %#v err=%v", match, err)
	}

	hc := NewHardcoverClientAt("https://example.test", "key", "ua")
	if _, err := hc.graphql(context.Background(), "query", map[string]any{"bad": func() {}}); err == nil {
		t.Fatal("expected hardcover marshal error")
	}
	if match, err := hc.Fetch(context.Background(), "999999999999999999999999999999999999999999999999"); err != nil || match != nil {
		t.Fatalf("hardcover overflow fetch = %#v err=%v", match, err)
	}

	gbStatus := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer gbStatus.Close()
	gb := NewGoogleBooksClientAt(gbStatus.URL, "key", "ua")
	gb.client = gbStatus.Client()
	if _, err := gb.Fetch(context.Background(), "zyTCAlFPjgYC"); err == nil {
		t.Fatal("expected google fetch status error")
	}

	grInvalid := NewGoodreadsClientAt("http://[::1", "ua")
	if _, err := grInvalid.Fetch(context.Background(), "123"); err == nil {
		t.Fatal("expected goodreads create request error")
	}
	grNoTitle := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<script type="application/ld+json">{"@type":"Book","url":"https://www.goodreads.com/book/show/123"}</script>`))
	}))
	defer grNoTitle.Close()
	grClient := NewGoodreadsClientAt(grNoTitle.URL, "ua")
	grClient.client = grNoTitle.Client()
	if match, err := grClient.Fetch(context.Background(), "123"); err != nil || match != nil {
		t.Fatalf("goodreads no-title fetch = %#v err=%v", match, err)
	}
	if parseGoodreadsJSONLD(`<script></script>`) != nil {
		t.Fatal("goodreads missing jsonld")
	}
}

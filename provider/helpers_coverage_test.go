package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prairie-server/prairie-plugin-metadata-ebook/metadata"
)

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

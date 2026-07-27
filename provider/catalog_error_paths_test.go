package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prairie-server/prairie-plugin-metadata-ebook/metadata"
)

func TestCatalogHTTPErrorAndFetchMissPaths(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "fail", http.StatusInternalServerError)
	}))
	defer srv.Close()

	q := metadata.SearchQuery{Title: "hail mary"}

	g := NewGutenbergClientAt(srv.URL, "ua")
	g.http.client = srv.Client()
	if _, err := g.Search(context.Background(), q); err == nil {
		t.Fatal("gutenberg search error")
	}
	if _, err := g.Fetch(context.Background(), "84"); err == nil {
		t.Fatal("gutenberg fetch error")
	}

	bb := NewBookBrainzClientAt(srv.URL, "ua")
	bb.http.client = srv.Client()
	if _, err := bb.Search(context.Background(), q); err == nil {
		t.Fatal("bookbrainz search error")
	}
	if _, err := bb.Fetch(context.Background(), bbValidID); err == nil {
		t.Fatal("bookbrainz fetch error")
	}

	ff := NewFantasticFictionClientAt(srv.URL, "ua")
	ff.http.client = srv.Client()
	if _, err := ff.Search(context.Background(), q); err == nil {
		t.Fatal("ff search error")
	}
	if _, err := ff.Fetch(context.Background(), "path:/andy-weir/project-hail-mary.htm"); err == nil {
		t.Fatal("ff fetch error")
	}

	isfdb := NewISFDBClientAt(srv.URL, "ua")
	isfdb.http.client = srv.Client()
	if _, err := isfdb.Search(context.Background(), q); err == nil {
		t.Fatal("isfdb search error")
	}
	if _, err := isfdb.Fetch(context.Background(), "12345"); err == nil {
		t.Fatal("isfdb fetch error")
	}

	lt := NewLibraryThingClientAt(srv.URL, "ua")
	lt.http.client = srv.Client()
	if _, err := lt.Search(context.Background(), q); err == nil {
		t.Fatal("lt search error")
	}
	if _, err := lt.Fetch(context.Background(), "work:12345"); err == nil {
		t.Fatal("lt fetch error")
	}

	ia := NewInternetArchiveClientAt(srv.URL, "ua")
	ia.http.client = srv.Client()
	if _, err := ia.Search(context.Background(), q); err == nil {
		t.Fatal("ia search error")
	}
	if _, err := ia.Fetch(context.Background(), "itemid"); err == nil {
		t.Fatal("ia fetch error")
	}

	wc := NewWorldCatClientAt(srv.URL, "ua")
	wc.http.client = srv.Client()
	if _, err := wc.Search(context.Background(), q); err == nil {
		t.Fatal("wc search error")
	}
	if _, err := wc.Fetch(context.Background(), "9780593135204"); err == nil {
		t.Fatal("wc fetch error")
	}

	db := NewDoubanClientAt(srv.URL, "ua")
	db.http.client = srv.Client()
	if _, err := db.Search(context.Background(), q); err == nil {
		t.Fatal("douban search error")
	}
	if _, err := db.Fetch(context.Background(), "1234567"); err == nil {
		t.Fatal("douban fetch error")
	}
}

func TestCatalogNotFoundFetchReturnsNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	ff := NewFantasticFictionClientAt(srv.URL, "ua")
	ff.http.client = srv.Client()
	if m, err := ff.Fetch(context.Background(), "path:/missing/book.htm"); err != nil || m != nil {
		t.Fatalf("ff 404: %v %#v", err, m)
	}
	isfdb := NewISFDBClientAt(srv.URL, "ua")
	isfdb.http.client = srv.Client()
	if m, err := isfdb.Fetch(context.Background(), "1"); err != nil || m != nil {
		t.Fatalf("isfdb 404: %v %#v", err, m)
	}
	lt := NewLibraryThingClientAt(srv.URL, "ua")
	lt.http.client = srv.Client()
	if m, err := lt.Fetch(context.Background(), "work:1"); err != nil || m != nil {
		t.Fatalf("lt 404: %v %#v", err, m)
	}
	ia := NewInternetArchiveClientAt(srv.URL, "ua")
	ia.http.client = srv.Client()
	if m, err := ia.Fetch(context.Background(), "x"); err != nil || m != nil {
		t.Fatalf("ia 404: %v %#v", err, m)
	}
	wc := NewWorldCatClientAt(srv.URL, "ua")
	wc.http.client = srv.Client()
	if m, err := wc.Fetch(context.Background(), "9780593135204"); err != nil || m != nil {
		t.Fatalf("wc 404: %v %#v", err, m)
	}
	db := NewDoubanClientAt(srv.URL, "ua")
	db.http.client = srv.Client()
	if m, err := db.Fetch(context.Background(), "1"); err != nil || m != nil {
		t.Fatalf("douban 404: %v %#v", err, m)
	}
	g := NewGutenbergClientAt(srv.URL, "ua")
	g.http.client = srv.Client()
	if m, err := g.Fetch(context.Background(), "1"); err != nil || m != nil {
		t.Fatalf("gutenberg 404: %v %#v", err, m)
	}
	bb := NewBookBrainzClientAt(srv.URL, "ua")
	bb.http.client = srv.Client()
	if m, err := bb.Fetch(context.Background(), bbValidID); err != nil || m != nil {
		t.Fatalf("bookbrainz 404: %v %#v", err, m)
	}
}

func TestGoodreadsHTTPErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "fail", http.StatusBadGateway)
	}))
	defer srv.Close()
	c := NewGoodreadsClientAt(srv.URL, "ua")
	c.client = srv.Client()
	if _, err := c.Search(context.Background(), metadata.SearchQuery{Title: "x"}); err == nil {
		t.Fatal("search error")
	}
	if _, err := c.Fetch(context.Background(), "123"); err == nil {
		t.Fatal("fetch error")
	}
}

func TestParseHelpersEdgeHTML(t *testing.T) {
	if parseFantasticFictionBookPage([]byte(`no title`)) != nil {
		t.Fatal("ff empty")
	}
	if parseWorldCatRecordPage([]byte(`nope`)) != nil {
		t.Fatal("wc empty")
	}
	if parseDoubanSubjectPage([]byte(`nope`)) != nil {
		t.Fatal("douban empty")
	}
	if len(parseDoubanSearchPage([]byte(`nope`))) != 0 {
		t.Fatal("douban search empty")
	}
	_ = htmlText("&#999999999;") // invalid entity path
	_ = aaStripText("&#0;")
	_ = aaFirstSubmatch(catalogNumericRE, "abc")
}

func TestCatalogParsersWithFixtures(t *testing.T) {
	if parseFantasticFictionBookPage(loadProviderFixture(t, "fantasticfiction_book.html")) == nil {
		t.Fatal("ff book")
	}
	if len(parseFantasticFictionSearchPage(loadProviderFixture(t, "fantasticfiction_search.html"))) == 0 {
		t.Fatal("ff search")
	}
	if parseISFDBTitlePage(loadProviderFixture(t, "isfdb_title.html")) == nil {
		t.Fatal("isfdb title")
	}
	if len(parseISFDBSearchPage(loadProviderFixture(t, "isfdb_search.html"))) == 0 {
		t.Fatal("isfdb search")
	}
	if parseLibraryThingWorkPage(loadProviderFixture(t, "librarything_work.html")) == nil {
		t.Fatal("lt work")
	}
	if len(parseLibraryThingSearchPage(loadProviderFixture(t, "librarything_search.html"))) == 0 {
		t.Fatal("lt search")
	}
	if parseWorldCatRecordPage(loadProviderFixture(t, "worldcat_record.html")) == nil {
		t.Fatal("wc record")
	}
	if len(parseWorldCatSearchPage(loadProviderFixture(t, "worldcat_search.html"))) == 0 {
		t.Fatal("wc search")
	}
	if parseDoubanSubjectPage(loadProviderFixture(t, "douban_subject.html")) == nil {
		t.Fatal("douban subject")
	}
	if len(parseDoubanSearchPage(loadProviderFixture(t, "douban_search.html"))) == 0 {
		t.Fatal("douban search")
	}
}

func TestOpenLibraryHardcoverISBNdbHTTPErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "fail", http.StatusBadGateway)
	}))
	defer srv.Close()

	ol := NewOpenLibraryClientAt(srv.URL, "https://covers.openlibrary.org", "ua")
	ol.client = srv.Client()
	if _, err := ol.Search(context.Background(), metadata.SearchQuery{Title: "x"}); err == nil {
		t.Fatal("ol search")
	}
	if _, err := ol.Fetch(context.Background(), "9780593135204"); err == nil {
		t.Fatal("ol fetch")
	}

	hc := NewHardcoverClientAt(srv.URL, "key", "ua")
	hc.client = srv.Client()
	if _, err := hc.Search(context.Background(), metadata.SearchQuery{Title: "x"}); err == nil {
		t.Fatal("hc search")
	}
	if _, err := hc.Fetch(context.Background(), "1"); err == nil {
		t.Fatal("hc fetch")
	}

	isbn := NewISBNdbClientAt(srv.URL, "key", "ua")
	isbn.client = srv.Client()
	if _, err := isbn.Search(context.Background(), metadata.SearchQuery{Title: "x"}); err == nil {
		t.Fatal("isbn search")
	}
	if _, err := isbn.Fetch(context.Background(), "9780593135204"); err == nil {
		t.Fatal("isbn fetch")
	}

	am := NewAmazonClientAt(srv.URL, "ua")
	am.client = srv.Client()
	if _, err := am.Search(context.Background(), metadata.SearchQuery{ProviderIDs: map[string]string{"amazon": "B08G9PRS1K"}}); err == nil {
		t.Fatal("amazon search")
	}
	if _, err := am.Fetch(context.Background(), "B08G9PRS1K"); err == nil {
		t.Fatal("amazon fetch")
	}
}

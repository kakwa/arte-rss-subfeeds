package main

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestServeCategoryFeedRoutesByLanguage(t *testing.T) {
	db, err := openDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}
	defer db.Close()

	items := []entry{
		{Language: "fr", GUID: "1", Title: "FR Histoire item", Link: "https://example.org/fr", Description: "desc fr", Category: "Histoire", RawCategory: "Histoire", PubDate: time.Now()},
		{Language: "de", GUID: "1", Title: "DE Geschichte item", Link: "https://example.org/de", Description: "desc de", Category: "Geschichte", RawCategory: "Geschichte", PubDate: time.Now()},
	}
	if _, err := storeItems(db, items); err != nil {
		t.Fatalf("storeItems: %v", err)
	}

	mux := http.NewServeMux()
	registerFeedRoutes(mux, db)

	// The fr and de feeds for the analogous category ("Histoire" / "Geschichte")
	// must be reachable under distinct, language-prefixed paths.
	frRes := doGet(t, mux, "/feeds/fr/histoire.rss")
	if got := decodeTitles(t, frRes.Body.Bytes()); len(got) != 1 || got[0] != "FR Histoire item" {
		t.Fatalf("fr/histoire.rss items = %v", got)
	}

	deRes := doGet(t, mux, "/feeds/de/geschichte.rss")
	if got := decodeTitles(t, deRes.Body.Bytes()); len(got) != 1 || got[0] != "DE Geschichte item" {
		t.Fatalf("de/geschichte.rss items = %v", got)
	}

	// Cross-language paths (fr content under /de/, and vice versa) must not exist.
	if res := doGet(t, mux, "/feeds/de/histoire.rss"); res.Code != http.StatusNotFound {
		t.Errorf("/feeds/de/histoire.rss status = %d, want 404", res.Code)
	}
	if res := doGet(t, mux, "/feeds/fr/geschichte.rss"); res.Code != http.StatusNotFound {
		t.Errorf("/feeds/fr/geschichte.rss status = %d, want 404", res.Code)
	}

	ct := frRes.Header().Get("Content-Type")
	if ct != "application/rss+xml; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestServeIndexListsBothLanguages(t *testing.T) {
	db, err := openDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	registerFeedRoutes(mux, db)

	res := doGet(t, mux, "/")
	body := res.Body.String()
	for _, want := range []string{"/feeds/fr/sciences.rss", "/feeds/de/wissenschaft.rss"} {
		if !strings.Contains(body, want) {
			t.Errorf("index page missing link %q", want)
		}
	}
}

func doGet(t *testing.T, mux *http.ServeMux, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	return res
}

func decodeTitles(t *testing.T, body []byte) []string {
	t.Helper()
	var out outRSS
	if err := xml.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode rss: %v", err)
	}
	titles := make([]string, len(out.Channel.Items))
	for i, it := range out.Channel.Items {
		titles[i] = it.Title
	}
	return titles
}

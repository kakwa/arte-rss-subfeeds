package main

import (
	"encoding/json"
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
	frRes := doGet(t, mux, "/feeds/fr/histoire.xml")
	if got := decodeTitles(t, frRes.Body.Bytes()); len(got) != 1 || got[0] != "FR Histoire item" {
		t.Fatalf("fr/histoire.xml items = %v", got)
	}

	deRes := doGet(t, mux, "/feeds/de/geschichte.xml")
	if got := decodeTitles(t, deRes.Body.Bytes()); len(got) != 1 || got[0] != "DE Geschichte item" {
		t.Fatalf("de/geschichte.xml items = %v", got)
	}

	// Cross-language paths (fr content under /de/, and vice versa) must not exist.
	if res := doGet(t, mux, "/feeds/de/histoire.xml"); res.Code != http.StatusNotFound {
		t.Errorf("/feeds/de/histoire.xml status = %d, want 404", res.Code)
	}
	if res := doGet(t, mux, "/feeds/fr/geschichte.xml"); res.Code != http.StatusNotFound {
		t.Errorf("/feeds/fr/geschichte.xml status = %d, want 404", res.Code)
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
	for _, want := range []string{"/feeds/fr/sciences.xml", "/feeds/de/wissenschaft.xml"} {
		if !strings.Contains(body, want) {
			t.Errorf("index page missing link %q", want)
		}
	}

	// The homepage always lists French categories before German ones.
	frPos := strings.Index(body, "/feeds/fr/sciences.xml")
	dePos := strings.Index(body, "/feeds/de/wissenschaft.xml")
	if frPos == -1 || dePos == -1 || frPos > dePos {
		t.Errorf("expected fr section before de section, frPos=%d dePos=%d", frPos, dePos)
	}
}

func TestServeCategoryPreviewJSON(t *testing.T) {
	db, err := openDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	registerFeedRoutes(mux, db)

	// No entries stored yet: must return an empty JSON array, not "null".
	res := doGet(t, mux, "/api/entries/fr/sciences")
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	if got := strings.TrimSpace(res.Body.String()); got != "[]" {
		t.Errorf("empty preview body = %q, want %q", got, "[]")
	}

	items := []entry{
		{Language: "fr", GUID: "1", Title: "Older", Link: "https://example.org/older", Category: "Sciences", RawCategory: "Sciences", PubDate: time.Now().Add(-time.Hour)},
		{Language: "fr", GUID: "2", Title: "Newer", Link: "https://example.org/newer", Category: "Sciences", RawCategory: "Sciences", PubDate: time.Now()},
		{Language: "fr", GUID: "3", Title: "TooOld", Link: "https://example.org/tooold", Category: "Sciences", RawCategory: "Sciences", PubDate: time.Now().Add(-31 * 24 * time.Hour)},
	}
	if _, err := storeItems(db, items); err != nil {
		t.Fatalf("storeItems: %v", err)
	}

	res = doGet(t, mux, "/api/entries/fr/sciences")
	var got []previewEntry
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode preview json: %v", err)
	}
	// TooOld is more than 30 days in the past and must be excluded.
	if len(got) != 2 || got[0].Title != "Newer" || got[1].Title != "Older" {
		t.Fatalf("preview entries = %+v, want [Newer, Older]", got)
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

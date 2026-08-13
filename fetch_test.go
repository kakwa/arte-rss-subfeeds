package main

import (
	"os"
	"testing"
	"time"
)

// The testdata/*_sample.rss fixtures are trimmed extracts of the real
// https://www.arte.tv/partnerFeeds/rss/schedule/today/{fr,de}.rss feeds,
// picked to cover one item per known category plus one item whose first
// <category> ("Séries et fictions" / "Fernsehfilme und Serien") is an
// alias resolved via categoryAliases rather than an exact match.

func TestParseFeedFR(t *testing.T) {
	body, err := os.ReadFile("testdata/fr_sample.rss")
	if err != nil {
		t.Fatal(err)
	}

	entries, err := parseFeed(body, "fr")
	if err != nil {
		t.Fatalf("parseFeed: %v", err)
	}
	if len(entries) != 6 {
		t.Fatalf("got %d entries, want 6", len(entries))
	}

	want := map[string]string{
		"5137193_120877-006-A": "Info et société",
		"5242981_090605-006-A": "Voyages et découvertes",
		"5137197_120086-111-A": "Culture et pop",
		"5137199_122158-000-A": "Histoire",
		"5233468_124442-000-A": "Sciences",
		"5137204_110221-000-A": "Séries", // raw category "Séries et fictions" aliases to "Séries"
	}

	for _, e := range entries {
		if e.Language != "fr" {
			t.Errorf("entry %s: language = %q, want fr", e.GUID, e.Language)
		}
		wantCat, ok := want[e.GUID]
		if !ok {
			t.Fatalf("unexpected guid %s", e.GUID)
		}
		if e.Category != wantCat {
			t.Errorf("guid %s: category = %q, want %q (raw=%q)", e.GUID, e.Category, wantCat, e.RawCategory)
		}
	}

	// Spot-check field decoding on the first item.
	first := entries[0]
	if first.Title == "" || first.Link == "" || first.Description == "" {
		t.Errorf("entry %+v has empty required field", first)
	}
	wantDate := time.Date(2026, time.August, 13, 5, 0, 0, 0, time.FixedZone("", 2*60*60))
	if !first.PubDate.Equal(wantDate) {
		t.Errorf("pubDate = %v, want %v", first.PubDate, wantDate)
	}
	if first.RawCategory != "Info et société" {
		t.Errorf("rawCategory = %q, want %q", first.RawCategory, "Info et société")
	}
}

func TestParseFeedDE(t *testing.T) {
	body, err := os.ReadFile("testdata/de_sample.rss")
	if err != nil {
		t.Fatal(err)
	}

	entries, err := parseFeed(body, "de")
	if err != nil {
		t.Fatalf("parseFeed: %v", err)
	}
	if len(entries) != 6 {
		t.Fatalf("got %d entries, want 6", len(entries))
	}

	want := map[string]string{
		"5242977_090605-006-A": "Entdeckung der Welt",
		"5137012_120086-111-A": "Kultur und Pop",
		"5137021_110221-000-A": "Serien", // raw category "Fernsehfilme und Serien" aliases to "Serien"
		"5137026_129043-162-A": "Aktuelles und Gesellschaft",
		"5137028_122158-000-A": "Geschichte",
		"5233467_124442-000-A": "Wissenschaft",
	}

	for _, e := range entries {
		if e.Language != "de" {
			t.Errorf("entry %s: language = %q, want de", e.GUID, e.Language)
		}
		wantCat, ok := want[e.GUID]
		if !ok {
			t.Fatalf("unexpected guid %s", e.GUID)
		}
		if e.Category != wantCat {
			t.Errorf("guid %s: category = %q, want %q (raw=%q)", e.GUID, e.Category, wantCat, e.RawCategory)
		}
	}
}

func TestParseFeedInvalidXML(t *testing.T) {
	if _, err := parseFeed([]byte("not xml"), "fr"); err == nil {
		t.Fatal("expected an error for invalid XML, got nil")
	}
}

func TestParseFeedSkipsUnparsablePubDate(t *testing.T) {
	const rss = `<?xml version="1.0"?>
<rss><channel>
<item><title>bad date</title><link>x</link><description>d</description><category>Sciences</category><guid>1</guid><pubDate>not-a-date</pubDate></item>
<item><title>good date</title><link>x</link><description>d</description><category>Sciences</category><guid>2</guid><pubDate>Thu, 13 Aug 2026 05:00:00 +0200</pubDate></item>
</channel></rss>`

	entries, err := parseFeed([]byte(rss), "fr")
	if err != nil {
		t.Fatalf("parseFeed: %v", err)
	}
	if len(entries) != 1 || entries[0].GUID != "2" {
		t.Fatalf("got %+v, want only the entry with a parsable pubDate", entries)
	}
}

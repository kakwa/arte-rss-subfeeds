package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStoreAndQueryByCategoryAndLanguage(t *testing.T) {
	db, err := openDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}
	defer db.Close()

	frItems := []entry{
		{Language: "fr", GUID: "1", Title: "FR Sciences", Link: "l1", Category: "Sciences", RawCategory: "Sciences", PubDate: time.Now()},
		{Language: "fr", GUID: "2", Title: "FR Divers", Link: "l2", Category: uncategorized, RawCategory: "Séries et fictions", PubDate: time.Now()},
	}
	deItems := []entry{
		{Language: "de", GUID: "1", Title: "DE Sciences", Link: "l3", Category: "Wissenschaft", RawCategory: "Wissenschaft", PubDate: time.Now()},
	}

	if _, err := storeItems(db, frItems); err != nil {
		t.Fatalf("storeItems fr: %v", err)
	}
	if _, err := storeItems(db, deItems); err != nil {
		t.Fatalf("storeItems de: %v", err)
	}

	// Same guid "1" exists for both fr and de: language must disambiguate.
	frSci, err := entriesByCategory(db, "fr", "Sciences", 10)
	if err != nil {
		t.Fatalf("entriesByCategory: %v", err)
	}
	if len(frSci) != 1 || frSci[0].Title != "FR Sciences" {
		t.Fatalf("fr Sciences = %+v, want a single FR Sciences entry", frSci)
	}

	deSci, err := entriesByCategory(db, "de", "Wissenschaft", 10)
	if err != nil {
		t.Fatalf("entriesByCategory: %v", err)
	}
	if len(deSci) != 1 || deSci[0].Title != "DE Sciences" {
		t.Fatalf("de Wissenschaft = %+v, want a single DE Sciences entry", deSci)
	}

	// A German query for an fr-only category must return nothing.
	deSciFromFrCat, err := entriesByCategory(db, "de", "Sciences", 10)
	if err != nil {
		t.Fatalf("entriesByCategory: %v", err)
	}
	if len(deSciFromFrCat) != 0 {
		t.Fatalf("de/Sciences = %+v, want no rows", deSciFromFrCat)
	}

	frDivers, err := entriesByCategory(db, "fr", uncategorized, 10)
	if err != nil {
		t.Fatalf("entriesByCategory: %v", err)
	}
	if len(frDivers) != 1 || frDivers[0].GUID != "2" {
		t.Fatalf("fr Divers = %+v, want the unmatched-category entry", frDivers)
	}
}

func TestStoreItemsUpsertsByLanguageAndGUID(t *testing.T) {
	db, err := openDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}
	defer db.Close()

	original := []entry{
		{Language: "fr", GUID: "42", Title: "Original title", Link: "l", Category: "Histoire", RawCategory: "Histoire", PubDate: time.Now()},
	}
	if _, err := storeItems(db, original); err != nil {
		t.Fatalf("storeItems: %v", err)
	}

	updated := []entry{
		{Language: "fr", GUID: "42", Title: "Updated title", Link: "l", Category: "Histoire", RawCategory: "Histoire", PubDate: time.Now()},
	}
	if _, err := storeItems(db, updated); err != nil {
		t.Fatalf("storeItems (update): %v", err)
	}

	got, err := entriesByCategory(db, "fr", "Histoire", 10)
	if err != nil {
		t.Fatalf("entriesByCategory: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d rows, want 1 (guid should be upserted, not duplicated)", len(got))
	}
	if got[0].Title != "Updated title" {
		t.Fatalf("title = %q, want %q", got[0].Title, "Updated title")
	}
}

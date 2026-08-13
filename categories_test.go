package main

import "testing"

func TestMapCategory(t *testing.T) {
	cases := []struct {
		lang string
		raw  string
		want string
	}{
		{"fr", "Sciences", "Sciences"},
		{"fr", "Voyages et découvertes", "Voyages et découvertes"},
		{"fr", "Séries et fictions", "Séries"},
		{"fr", "Enquêtes et reportages", "Documentaires et reportages"},
		{"fr", "Evasion", "Voyages et découvertes"},
		{"fr", "Médecine et santé", "Sciences"},
		{"fr", "Arts", "Culture et pop"},
		{"fr", "XXe siècle", "Histoire"},
		{"fr", "", uncategorized},
		{"de", "Wissenschaft", "Wissenschaft"},
		{"de", "Entdeckung der Welt", "Entdeckung der Welt"},
		{"de", "Fernsehfilme und Serien", "Serien"},
		{"de", "Aktuelles", "Aktuelles und Gesellschaft"},
		{"de", "Das 20. Jahrhundert", "Geschichte"},
		{"de", "Gesundheit und Medizin", "Wissenschaft"},
		{"de", "Kunst", "Kultur und Pop"},
		{"de", "Reisen", "Entdeckung der Welt"},
		{"de", "Sciences", uncategorized}, // fr category name is not valid for de
	}
	for _, c := range cases {
		if got := mapCategory(c.lang, c.raw); got != c.want {
			t.Errorf("mapCategory(%q, %q) = %q, want %q", c.lang, c.raw, got, c.want)
		}
	}
}

func TestAllCategoriesIncludesDivers(t *testing.T) {
	for _, lang := range []string{"fr", "de"} {
		cats := allCategories(lang)
		if len(cats) != len(knownCategories[lang])+1 {
			t.Fatalf("%s: got %d categories, want %d", lang, len(cats), len(knownCategories[lang])+1)
		}
		if cats[len(cats)-1] != uncategorized {
			t.Errorf("%s: last category = %q, want %q", lang, cats[len(cats)-1], uncategorized)
		}
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Voyages et découvertes": "voyages-et-decouvertes",
		"Séries":                 "series",
		"ARTE Concert":           "arte-concert",
		"Entdeckung der Welt":    "entdeckung-der-welt",
		"Info et société":        "info-et-societe",
		"Divers":                 "divers",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

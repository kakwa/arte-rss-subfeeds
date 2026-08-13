package main

import "strings"

// knownCategories is the fixed list of categories we split entries into per
// language, matched against the first <category> tag of each feed item.
var knownCategories = map[string][]string{
	"fr": {
		"Documentaires et reportages",
		"Films",
		"Séries",
		"Info et société",
		"Culture et pop",
		"ARTE Concert",
		"Sciences",
		"Voyages et découvertes",
		"Histoire",
	},
	"de": {
		"Dokus und Reportagen",
		"Filme",
		"Serien",
		"Aktuelles und Gesellschaft",
		"Kultur und Pop",
		"ARTE Concert",
		"Wissenschaft",
		"Entdeckung der Welt",
		"Geschichte",
	},
}

const uncategorized = "Divers"

// categoryAliases maps alternate labels ARTE uses for the same category
// (e.g. a longer or shorter variant) to the canonical name in
// knownCategories, so they don't fall through to "Divers".
var categoryAliases = map[string]map[string]string{
	"fr": {
		"Séries et fictions":     "Séries",
		"Enquêtes et reportages": "Documentaires et reportages",
		"Evasion":                "Voyages et découvertes",
		"Médecine et santé":      "Sciences",
		"Arts":                   "Culture et pop",
		"XXe siècle":             "Histoire",
	},
	"de": {
		"Fernsehfilme und Serien": "Serien",
		"Aktuelles":               "Aktuelles und Gesellschaft",
		"Das 20. Jahrhundert":     "Geschichte",
		"Gesundheit und Medizin":  "Wissenschaft",
		"Kunst":                   "Kultur und Pop",
		"Reisen":                  "Entdeckung der Welt",
	},
}

var categorySets = func() map[string]map[string]bool {
	sets := make(map[string]map[string]bool, len(knownCategories))
	for lang, cats := range knownCategories {
		set := make(map[string]bool, len(cats))
		for _, c := range cats {
			set[c] = true
		}
		sets[lang] = set
	}
	return sets
}()

// allCategories returns the known categories for a language plus the
// catch-all "Divers".
func allCategories(lang string) []string {
	known := knownCategories[lang]
	cats := make([]string, 0, len(known)+1)
	cats = append(cats, known...)
	cats = append(cats, uncategorized)
	return cats
}

// mapCategory maps a feed's first <category> value to one of the known
// categories for that language, or to "Divers" if it doesn't match any of them.
func mapCategory(lang, raw string) string {
	if categorySets[lang][raw] {
		return raw
	}
	if canonical, ok := categoryAliases[lang][raw]; ok {
		return canonical
	}
	return uncategorized
}

var slugReplacer = strings.NewReplacer(
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"à", "a", "â", "a",
	"î", "i", "ï", "i",
	"ô", "o",
	"û", "u", "ù", "u", "ü", "u",
	"ç", "c",
	" ", "-",
)

// slugify turns a category name into a URL-safe slug, e.g.
// "Voyages et découvertes" -> "voyages-et-decouvertes".
func slugify(s string) string {
	s = slugReplacer.Replace(strings.ToLower(s))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

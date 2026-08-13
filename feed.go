package main

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"net/http"
	"time"
)

const maxFeedItems = 200

type outRSS struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel outChannel `xml:"channel"`
}

type outChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Language    string    `xml:"language"`
	Items       []outItem `xml:"item"`
}

type outItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description cdata  `xml:"description"`
	Category    string `xml:"category"`
	GUID        string `xml:"guid"`
	PubDate     string `xml:"pubDate"`
}

type cdata struct {
	Text string `xml:",cdata"`
}

// supportedLanguages lists the languages served, each with its guide page
// link used in generated feed channels.
var supportedLanguages = map[string]string{
	"fr": "https://www.arte.tv/fr/guide/",
	"de": "https://www.arte.tv/de/guide/",
}

func registerFeedRoutes(mux *http.ServeMux, db *sql.DB) {
	for lang := range supportedLanguages {
		for _, cat := range allCategories(lang) {
			lang, cat := lang, cat
			mux.HandleFunc("/feeds/"+lang+"/"+slugify(cat)+".rss", func(w http.ResponseWriter, r *http.Request) {
				serveCategoryFeed(w, db, lang, cat)
			})
		}
	}
	mux.HandleFunc("/", serveIndex)
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, "<h1>ARTE category feeds</h1>")
	for _, lang := range []string{"fr", "de"} {
		fmt.Fprintf(w, "<h2>%s</h2><ul>", lang)
		for _, cat := range allCategories(lang) {
			fmt.Fprintf(w, `<li><a href="/feeds/%s/%s.rss">%s</a></li>`, lang, slugify(cat), cat)
		}
		fmt.Fprint(w, "</ul>")
	}
}

func serveCategoryFeed(w http.ResponseWriter, db *sql.DB, lang, category string) {
	items, err := entriesByCategory(db, lang, category, maxFeedItems)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	out := outRSS{
		Version: "2.0",
		Channel: outChannel{
			Title:       "ARTE Programme TV - " + category,
			Link:        supportedLanguages[lang],
			Description: "Programmes ARTE - catégorie " + category,
			Language:    lang,
		},
	}
	for _, it := range items {
		out.Channel.Items = append(out.Channel.Items, outItem{
			Title:       it.Title,
			Link:        it.Link,
			Description: cdata{Text: it.Description},
			Category:    it.RawCategory,
			GUID:        it.GUID,
			PubDate:     it.PubDate.Format(time.RFC1123Z),
		})
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(out); err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
	}
}

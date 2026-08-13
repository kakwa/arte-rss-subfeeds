package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type rssFeed struct {
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	Description string   `xml:"description"`
	Categories  []string `xml:"category"`
	GUID        string   `xml:"guid"`
	PubDate     string   `xml:"pubDate"`
}

type entry struct {
	Language    string
	GUID        string
	Title       string
	Link        string
	Description string
	RawCategory string
	Category    string
	PubDate     time.Time
}

// fetchFeed downloads and parses the ARTE schedule RSS feed for a language.
func fetchFeed(url, lang string) ([]entry, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseFeed(body, lang)
}

// parseFeed decodes RSS body bytes into entries, mapping each item's first
// <category> tag to one of the known categories for lang (or "Divers").
func parseFeed(body []byte, lang string) ([]entry, error) {
	var feed rssFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, err
	}

	entries := make([]entry, 0, len(feed.Channel.Items))
	for _, it := range feed.Channel.Items {
		pubDate, err := time.Parse(time.RFC1123Z, strings.TrimSpace(it.PubDate))
		if err != nil {
			continue
		}
		raw := ""
		if len(it.Categories) > 0 {
			raw = strings.TrimSpace(it.Categories[0])
		}
		entries = append(entries, entry{
			Language:    lang,
			GUID:        it.GUID,
			Title:       it.Title,
			Link:        it.Link,
			Description: it.Description,
			RawCategory: raw,
			Category:    mapCategory(lang, raw),
			PubDate:     pubDate,
		})
	}
	return entries, nil
}

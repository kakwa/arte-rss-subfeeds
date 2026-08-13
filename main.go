package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const feedURLTemplate = "https://www.arte.tv/partnerFeeds/rss/schedule/today/%s.rss"

func main() {
	dbPath := getEnv("DB_PATH", "arte.db")
	addr := getEnv("LISTEN_ADDR", ":8080")
	interval := getEnvDuration("FETCH_INTERVAL", 4*time.Hour)
	feedURLs := map[string]string{
		"fr": getEnv("FR_FEED_URL", fmt.Sprintf(feedURLTemplate, "fr")),
		"de": getEnv("DE_FEED_URL", fmt.Sprintf(feedURLTemplate, "de")),
	}

	db, err := openDB(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go runFetchLoop(ctx, db, feedURLs, interval)

	mux := http.NewServeMux()
	registerFeedRoutes(mux, db)
	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	log.Printf("listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}

func runFetchLoop(ctx context.Context, db *sql.DB, feedURLs map[string]string, interval time.Duration) {
	fetchAll(db, feedURLs)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fetchAll(db, feedURLs)
		}
	}
}

func fetchAll(db *sql.DB, feedURLs map[string]string) {
	for lang, url := range feedURLs {
		fetchOnce(db, url, lang)
	}
}

func fetchOnce(db *sql.DB, feedURL, lang string) {
	log.Printf("fetching %s (%s)", feedURL, lang)
	items, err := fetchFeed(feedURL, lang)
	if err != nil {
		log.Printf("fetch error (%s): %v", lang, err)
		return
	}
	n, err := storeItems(db, items)
	if err != nil {
		log.Printf("store error (%s): %v", lang, err)
		return
	}
	log.Printf("stored/updated %d entries (%s)", n, lang)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		log.Printf("invalid %s=%q, using default %s", key, v, fallback)
	}
	return fallback
}

package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const feedURLTemplate = "https://www.arte.tv/partnerFeeds/rss/schedule/today/%s.rss"

func main() {
	dbPath := flag.String("db-path", getEnv("DB_PATH", "arte.db"), "path to the sqlite database file")
	listen := flag.String("listen", getEnv("LISTEN", "127.0.0.1:8080"), "address to listen on, e.g. 127.0.0.1:8080")
	logLevel := flag.String("log-level", getEnv("LOG_LEVEL", "info"), "log level: debug, info, warn, error")
	flag.Parse()

	logger, err := newLogger(*logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	addr := *listen
	interval := getEnvDuration(logger, "FETCH_INTERVAL", 4*time.Hour)
	feedURLs := map[string]string{
		"fr": getEnv("FR_FEED_URL", fmt.Sprintf(feedURLTemplate, "fr")),
		"de": getEnv("DE_FEED_URL", fmt.Sprintf(feedURLTemplate, "de")),
	}

	db, err := openDB(*dbPath)
	if err != nil {
		logger.Fatal("open db", zap.Error(err))
	}
	defer db.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go runFetchLoop(ctx, logger, db, feedURLs, interval)

	mux := http.NewServeMux()
	registerFeedRoutes(mux, db)
	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	logger.Info("listening", zap.String("addr", addr))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("server", zap.Error(err))
	}
}

func newLogger(levelStr string) (*zap.Logger, error) {
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(levelStr)); err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", levelStr, err)
	}
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(level)
	cfg.Encoding = "console"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	return cfg.Build()
}

func runFetchLoop(ctx context.Context, logger *zap.Logger, db *sql.DB, feedURLs map[string]string, interval time.Duration) {
	fetchAll(logger, db, feedURLs)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fetchAll(logger, db, feedURLs)
		}
	}
}

func fetchAll(logger *zap.Logger, db *sql.DB, feedURLs map[string]string) {
	for lang, url := range feedURLs {
		fetchOnce(logger, db, url, lang)
	}
}

func fetchOnce(logger *zap.Logger, db *sql.DB, feedURL, lang string) {
	logger.Info("fetching", zap.String("url", feedURL), zap.String("lang", lang))
	items, err := fetchFeed(feedURL, lang)
	if err != nil {
		logger.Error("fetch error", zap.String("lang", lang), zap.Error(err))
		return
	}
	n, err := storeItems(db, items)
	if err != nil {
		logger.Error("store error", zap.String("lang", lang), zap.Error(err))
		return
	}
	logger.Info("stored/updated entries", zap.Int("count", n), zap.String("lang", lang))
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvDuration(logger *zap.Logger, key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		logger.Warn("invalid duration, using default", zap.String("key", key), zap.String("value", v), zap.Duration("default", fallback))
	}
	return fallback
}

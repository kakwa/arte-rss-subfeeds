package main

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// modernc.org/sqlite serializes access itself; a single connection
	// avoids "database is locked" errors between the fetch loop and HTTP reads.
	db.SetMaxOpenConns(1)

	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA busy_timeout=5000;",
	} {
		if _, err := db.Exec(pragma); err != nil {
			return nil, err
		}
	}

	const schema = `
	CREATE TABLE IF NOT EXISTS entries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		language TEXT NOT NULL,
		guid TEXT NOT NULL,
		title TEXT NOT NULL,
		link TEXT NOT NULL,
		description TEXT,
		raw_category TEXT,
		category TEXT NOT NULL,
		pub_date DATETIME NOT NULL,
		fetched_at DATETIME NOT NULL,
		UNIQUE(language, guid)
	);
	CREATE INDEX IF NOT EXISTS idx_entries_lang_category ON entries(language, category);
	CREATE INDEX IF NOT EXISTS idx_entries_pub_date ON entries(pub_date);
	`
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}
	return db, nil
}

// storeItems upserts entries by guid, so re-fetching the same item (e.g.
// a program still listed on the next run) just refreshes its row.
func storeItems(db *sql.DB, items []entry) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO entries (language, guid, title, link, description, raw_category, category, pub_date, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(language, guid) DO UPDATE SET
			title=excluded.title,
			link=excluded.link,
			description=excluded.description,
			raw_category=excluded.raw_category,
			category=excluded.category,
			pub_date=excluded.pub_date,
			fetched_at=excluded.fetched_at
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	now := time.Now().UTC()
	for _, it := range items {
		if _, err := stmt.Exec(it.Language, it.GUID, it.Title, it.Link, it.Description, it.RawCategory, it.Category, it.PubDate.UTC(), now); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(items), nil
}

func entriesByCategory(db *sql.DB, lang, category string, limit int) ([]entry, error) {
	rows, err := db.Query(`
		SELECT guid, title, link, description, raw_category, category, pub_date
		FROM entries
		WHERE language = ? AND category = ?
		ORDER BY pub_date DESC
		LIMIT ?
	`, lang, category, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntries(rows, lang)
}

// entriesByCategorySince is like entriesByCategory but excludes entries
// published before since, used for the homepage preview so it only ever
// shows recent programmes.
func entriesByCategorySince(db *sql.DB, lang, category string, since time.Time, limit int) ([]entry, error) {
	rows, err := db.Query(`
		SELECT guid, title, link, description, raw_category, category, pub_date
		FROM entries
		WHERE language = ? AND category = ? AND pub_date >= ?
		ORDER BY pub_date DESC
		LIMIT ?
	`, lang, category, since.UTC(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntries(rows, lang)
}

func scanEntries(rows *sql.Rows, lang string) ([]entry, error) {
	var out []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.GUID, &e.Title, &e.Link, &e.Description, &e.RawCategory, &e.Category, &e.PubDate); err != nil {
			return nil, err
		}
		e.Language = lang
		out = append(out, e)
	}
	return out, rows.Err()
}

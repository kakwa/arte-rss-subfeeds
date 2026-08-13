# arte-rss-subfeeds

RSS feed proxy for [arte.tv](https://www.arte.tv/).

Every 4 hours, it polls ARTE's daily programme feeds ([French](https://arte.tv/partnerFeeds/rss/schedule/today/fr.rss)
and [German](https://arte.tv/partnerFeeds/rss/schedule/today/de.rss)), records
the entries in a local SQLite database and re-exposes them as more convinient
per category (History, Voyage, Info, etc) RSS feeds.

Live instance available at **[arte-rss.kakwalab.ovh](https://arte-rss.kakwalab.ovh/)**:

## Build & test

```sh
make build      # builds ./arte-rss-subfeeds
make test       # go test ./...
make coverage   # test coverage report
```

## Run

```sh
./arte-rss-subfeeds --db-path arte.db --listen 127.0.0.1:8080 --log-level info
```

Flags (or equivalent env vars):

| Flag         | Env var    | Default             |
|--------------|------------|---------------------|
| `--db-path`  | `DB_PATH`  | `arte.db`           |
| `--listen`   | `LISTEN`   | `127.0.0.1:8080`    |
| `--log-level`| `LOG_LEVEL`| `info`              |

Additional env vars: `FETCH_INTERVAL` (default `4h`), `FR_FEED_URL`, `DE_FEED_URL`.

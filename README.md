# Ponisha + Karlancer Project Scraper (Go)

implementation of the Ponisha + Karlancer scraper.

## Features
- Parallel fan-out scrapers (goroutines) with per-provider error isolation
- High-budget filtering (>= 99,000,000 tomans by default; configurable in code) and DB deduplication via upsert
- Telegram alerts with queueing and rate limiting
- Cron schedule every 7 minutes
- Manual trigger endpoint: `GET /scraping`

## Requirements
- Go 1.23+
- Postgres
- sqlc

## Configuration
Copy `.env.example` and set values:

```
cp .env.example .env
```

Env vars:
- `DB_HOST`, `DB_PORT`, `DB_USERNAME`, `DB_PASSWORD`, `DB_DATABASE`, `DB_SSLMODE`
- `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID`, `TELEGRAM_CHAT_THREAD_ID` (optional)
- `HTTP_PORT`, `SCRAPE_CRON`

## Database Schema
Schema is in `db/schema.sql`. It is applied on startup.

The `projects` table uses a unique index on `(source, external_id)`.

## sqlc
Queries live in `db/queries.sql`. Generate code with:

```
sqlc generate
```

## Run
```
go mod tidy
sqlc generate

go run ./cmd/server
```

## Docker
```
docker-compose up -d
```

- Postgres host port: `5433`
- App port: `3000`
- The app waits for Postgres readiness via a compose healthcheck.

## Manual Trigger
```
curl http://localhost:3000/scraping
```

## Project Structure
Core packages:
- `cmd/server` entrypoint
- `internal/app` builder + lifecycle
- `internal/services/scraping` scrape orchestration
- `internal/providers/*` site scrapers
- `internal/repositories/sqlc` DB repository

## Dependency Injection
The app uses a builder pattern (`internal/app`) to compose dependencies. This makes it easy to swap
repositories, scrapers, or notifiers for tests and different environments.

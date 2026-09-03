# Demand Pricing Go

An explainable, deterministic demand-and-pricing decision-support demo for a fictional DIY retailer. It is a portfolio project: all data is synthetic, the rules are deliberately transparent, and its recommendations are **not** commercially validated pricing advice.

## What it demonstrates

- Go modular-monolith architecture with explicit PostgreSQL/`pgx` repositories.
- Idempotent six-column sales CSV ingestion with row-level errors.
- A transparent 7/28-day baseline forecast and confidence score.
- Explainable five-candidate price recommendations with promotion, history, price-bound, and margin-floor guards.
- PostgreSQL migrations, Prometheus metrics, health/readiness probes, graceful shutdown, Docker Compose, and CI.

## Quick start

```bash
docker compose -f deploy/docker-compose.yml up --build
```

Compose starts PostgreSQL, applies the migration, loads five stores, fifty products, inventory, and promotions, then exposes the API at `http://localhost:8080`.

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/api/v1/stores
curl 'http://localhost:8080/api/v1/products?limit=5'
```

## Local workflow

```bash
go test ./...
go vet ./...
go run ./cmd/seed --load-db
go run ./cmd/ingest --file ./db/seeds/daily_sales.csv
go run ./cmd/api
```

Set `DATABASE_URL` for local PostgreSQL, or copy `.env.example`. `go run ./cmd/seed` without `--load-db` only regenerates the reproducible CSV fixture.

## Data flow

```text
seed CSV -> ingest -> daily_sales -> forecast -> price recommendation -> API/dashboard
                           ^              |             |
                    stores/products  inventory      reason codes
```

The seed set intentionally covers high-stock declining demand, low-stock growing demand, an active promotion, a margin-floor case, and insufficient history.

## Pricing policy

The forecast is `0.6 * avg_7d + 0.4 * avg_28d`; confidence is non-promotion observations divided by 28, capped at 1. The optimizer evaluates -10%, -5%, current, +3%, and +5% candidates, clamps each to product bounds and the cost-plus-minimum-margin floor, and keeps the current price for active promotions, fewer than fourteen valid observations, or an improvement below 1%.

The API reference is in [docs/api.md](docs/api.md). Architecture and explicit trade-offs are in [docs/architecture.md](docs/architecture.md) and [docs/decisions.md](docs/decisions.md).

## Role alignment

This project maps to demand/pricing data products, REST APIs, ETL-style ingestion, PostgreSQL modelling, deterministic decision systems, and delivery/observability fundamentals. For production scale it would add stronger migration integration tests, background job orchestration, authentication, data-quality monitoring, and business validation; those are intentionally outside this compact demo.

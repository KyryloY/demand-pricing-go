# Demand Pricing Go Design

## Goal

Deliver a portfolio-grade, deterministic demand and pricing decision-support product for a fictional multi-store DIY retailer. It exposes an API and a small dashboard; it does not claim ML or production commercial suitability.

## Architecture

The system is a Go modular monolith. `cmd/api`, `cmd/ingest`, and `cmd/seed` are separate executables. HTTP handlers call application services through narrow interfaces; services own business workflows; PostgreSQL repositories own SQL and `pgx` interaction. The domain packages contain independently testable demand forecast and price recommendation rules.

## Data and persistence

PostgreSQL 16 holds stores, products, daily sales, inventory snapshots, promotions, forecasts, and recommendations. Money uses `NUMERIC(12,2)` in PostgreSQL and a decimal-safe Go representation. Migrations define lookup indexes and a unique promotion `code`, allowing `promotion_code` in the sales CSV to resolve reliably.

The seed command writes reproducible sample records from a fixed random seed and a ready-import CSV under `db/seeds/`. It deliberately creates high-stock declining demand, low-stock growing demand, active-promotion, margin-floor, and insufficient-history scenarios.

## Workflows

The CSV importer accepts the documented six columns, validates each row, resolves store/product/promotion identities, processes bounded concurrent chunks, and upserts in transactions. Valid rows remain committed when malformed rows are rejected; the final structured summary reports accepted, rejected, and upserted rows.

Forecasting derives 7-day and 28-day averages from non-promotion history when possible, applies the weighted baseline formula, calculates confidence from observation count, and stores an idempotent daily forecast. Recommendation calculation reads the latest price and inventory, enforces promotion/history/margin/price guards, evaluates the five specified candidate prices, and stores explanation-ready reason codes.

`POST /api/v1/forecasts/recalculate` and `POST /api/v1/recommendations/recalculate` accept JSON `{ "store_code", "sku", "date"? }`. `POST /api/v1/imports/sales` accepts a multipart CSV part. The web dashboard consumes the same services and offers store/product selection, summary signals, reasons, and a simple 28-day chart.

## Deterministic policy decisions

- Currency is EUR.
- The latest `daily_sales.unit_price` for a store/product is the current price.
- A gross-margin improvement below 1% is negligible and keeps the current price.
- Stock/trend pressure adds a small, documented 2% selection score bias toward the relevant lower or higher candidates; guardrails always win.
- Active promotions result in `promotion_active` and retain the current price.

## Reliability and operations

The API supplies request IDs, panic recovery, JSON structured request logs, `/healthz`, database-backed `/readyz`, Prometheus `/metrics`, database context deadlines, server timeouts, and graceful signal shutdown. Docker Compose runs migrations, PostgreSQL, and the API. GitHub Actions runs formatting, `go vet`, unit and PostgreSQL integration tests, plus Docker build.

## Testing

Tests are written first. Pure domain rules use fast unit tests. Repository, migration, import-idempotency, and representative HTTP behavior use PostgreSQL integration tests. Tests cover every stated guard rule and CSV validation outcome.

## Scope boundaries

No authentication, real retailer integrations, ORM, message broker, cache, SPA, ML training, Kubernetes, or cloud deployment is introduced. The README explicitly labels all data and recommendations synthetic and deterministic.

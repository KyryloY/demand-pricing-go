# Architecture

The service is a Go modular monolith. `cmd/api`, `cmd/ingest`, and `cmd/seed` are composition roots. Pure demand and pricing rules live in `internal/domain`; application workflows live in `internal/service`; PostgreSQL SQL remains in `internal/repository`; HTTP and operational concerns live in `internal/transport`, `internal/app`, and `internal/observability`.

PostgreSQL holds catalog, sales, inventory, promotions, forecasts, and recommendations. Each daily sales, inventory, forecast, and recommendation record has a store/product/date uniqueness constraint; lookup indexes are date-descending. Money uses `NUMERIC(12,2)` in PostgreSQL and decimal-safe types in Go.

The API runs migrations at startup after PostgreSQL is ready. Docker Compose first executes the deterministic seed loader, then starts the API. The importer and seed command can also run independently for local workflows.

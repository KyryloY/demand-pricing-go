# Demand Pricing Go Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (- [ ]) syntax for tracking.

**Goal:** Build the deterministic Go/PostgreSQL demand-pricing decision-support product specified in the approved design.

**Architecture:** A modular Go monolith keeps pure demand and pricing rules in internal/domain; application workflows in internal/service; and all pgx/SQL details in internal/repository. The API, importer, seed generator, dashboard, migrations, Docker Compose and CI share these bounded modules.

**Tech Stack:** Go 1.23+, chi, pgx/v5, PostgreSQL 16, golang-migrate, shopspring/decimal, slog, Prometheus client, Docker Compose, GitHub Actions.

**Spec:** docs/superpowers/specs/2026-09-02-demand-pricing-go-design.md; demand-pricing-go-spec.md

## Global Constraints

- Go 1.23+, net/http and chi; explicit SQL through pgx, never an ORM.
- PostgreSQL money is EUR NUMERIC(12,2); Go uses decimal-safe values.
- No auth, external retailer APIs, queues, cache, SPA, ML, Kubernetes, or cloud scope.
- Forecasts and recommendations are deterministic and explainable; tests precede production code.
- Import uses multipart HTTP and file-path CLI; it is idempotent and row-error aware.

---

## File map

- cmd/{api,ingest,seed}: executable composition roots.
- internal/domain/{catalog,sales,demand,pricing}: pure types and rules.
- internal/{repository,service,transport/http,observability,app}: data, workflows, API, operations and composition.
- db/{migrations,seeds}, web/{templates,static}, deploy, .github/workflows, docs: delivery assets.

### Task 1: Bootstrap PostgreSQL and developer environment

**Files:**
- Create: go.mod, .env.example, Makefile, deploy/Dockerfile, deploy/docker-compose.yml.
- Create: db/migrations/000001_initial_schema.up.sql, db/migrations/000001_initial_schema.down.sql, db/migrations/migrations_test.go.

**Produces:** all seven specified tables, a unique promotions.code, foreign keys, specified unique constraints and descending lookup indexes.

- [ ] **Step 1: Write failing migration test**

~~~go
func TestInitialMigrationCreatesPriceRecommendations(t *testing.T) {
    db := testpostgres.Open(t)
    migrateUp(t, db, "../../db/migrations")
    requireTable(t, db, "price_recommendations")
}
~~~

- [ ] **Step 2: Verify red**

Run: go test ./db/migrations -run TestInitialMigrationCreatesPriceRecommendations

Expected: FAIL because migration support is absent.

- [ ] **Step 3: Implement schema and Compose**

Add module dependencies (pgx, migrate, decimal, chi, Prometheus), the table schema/indexes/constraints, migration runner, Dockerfile and Compose PostgreSQL service.

- [ ] **Step 4: Verify green**

Run: go test ./db/migrations

Expected: PASS when INTEGRATION_DATABASE_URL is configured; explicitly skip only when it is absent.

- [ ] **Step 5: Commit**

~~~text
git add go.mod go.sum .env.example Makefile deploy db/migrations
git commit -m "chore: bootstrap Go service and database schema"
~~~

### Task 2: Domain contracts and reproducible seed data

**Files:**
- Create: internal/domain/{catalog,sales,demand,pricing}/*.go, internal/seed/generator.go, internal/seed/generator_test.go, cmd/seed/main.go, db/seeds/daily_sales.csv.

**Produces:** seed.Generate(w io.Writer, start time.Time) error and fixtures for 5 stores, 50 products, 120 sales dates and all five required business scenarios.

- [ ] **Step 1: Write failing fixed-seed test**

~~~go
func TestGenerateIsReproducible(t *testing.T) {
    var left, right bytes.Buffer
    require.NoError(t, Generate(&left, startDate))
    require.NoError(t, Generate(&right, startDate))
    require.Equal(t, left.String(), right.String())
}
~~~

- [ ] **Step 2: Verify red**

Run: go test ./internal/seed -run TestGenerateIsReproducible

Expected: FAIL because Generate does not exist.

- [ ] **Step 3: Implement minimum generator**

Use a fixed random seed and named fixtures for high-stock decline, low-stock growth, active-promotion, margin-floor and insufficient-history SKUs. Generate the documented six-column CSV and populate stores/products/promotions/inventory in the seed command.

- [ ] **Step 4: Verify green**

Run: go test ./internal/seed && go run ./cmd/seed --csv-out ./db/seeds/daily_sales.csv

Expected: PASS and a stable committed CSV.

- [ ] **Step 5: Commit**

~~~text
git add internal/domain internal/seed cmd/seed db/seeds
git commit -m "feat: add deterministic retail seed data"
~~~

### Task 3: CSV import and sales repositories

**Files:**
- Create: internal/repository/{catalog_repository,sales_repository}.go, internal/service/importer.go, internal/service/importer_test.go, cmd/ingest/main.go.

**Produces:** SalesImporter.Import(ctx, input) (ImportSummary, error); summary contains accepted/rejected/upserted counts and row-numbered errors.

- [ ] **Step 1: Write failing malformed-row test**

~~~go
func TestImportKeepsValidRowsWhenOneRowIsInvalid(t *testing.T) {
    summary, err := importer.Import(context.Background(), strings.NewReader(csvWithOneBadDate))
    require.NoError(t, err)
    require.Equal(t, 1, summary.Accepted)
    require.Equal(t, 1, summary.Rejected)
    require.Equal(t, 3, summary.Errors[0].Row)
}
~~~

- [ ] **Step 2: Verify red**

Run: go test ./internal/service -run TestImportKeepsValidRowsWhenOneRowIsInvalid

Expected: FAIL because importer is absent.

- [ ] **Step 3: Implement parser, batch transaction, and upsert**

Validate exact headers and every field, resolve store/SKU/promotion codes, parse with a bounded worker pool, batch valid entries into transactions, and use INSERT ON CONFLICT (store_id, product_id, sale_date) DO UPDATE.

- [ ] **Step 4: Verify green**

Run: go test ./internal/service ./internal/repository -run TestImport

Expected: PASS for header validation, row errors and importing the same CSV twice.

- [ ] **Step 5: Commit**

~~~text
git add internal/repository internal/service/importer.go internal/service/importer_test.go cmd/ingest
git commit -m "feat: import sales CSV idempotently"
~~~

### Task 4: Forecast calculation and persistence

**Files:**
- Create: internal/domain/demand/{calculator.go,calculator_test.go}, internal/repository/forecast_repository.go, internal/service/{forecast_service.go,forecast_service_test.go}.

**Produces:** demand.Calculate(asOf, observations) (Forecast, error) and ForecastService.Recalculate(ctx, storeCode, sku, date).

- [ ] **Step 1: Write failing forecast tests**

~~~go
func TestCalculateUsesWeightedSevenAndTwentyEightDayAverages(t *testing.T) {
    got, err := Calculate(day28, observations)
    require.NoError(t, err)
    require.InDelta(t, expectedForecast, got.DailyUnits, 0.001)
}
func TestCalculateConfidenceUsesNonPromotionObservationCount(t *testing.T) {
    got, _ := Calculate(day28, fourteenNonPromotionDays())
    require.Equal(t, 0.5, got.Confidence)
}
~~~

- [ ] **Step 2: Verify red**

Run: go test ./internal/domain/demand -run TestCalculate

Expected: FAIL because Calculate is absent.

- [ ] **Step 3: Implement formula and daily upsert**

Use non-promotion observations when enough exist; calculate 7/28 day means, 0.6*avg7 + 0.4*avg28, trend and min(1,count/28); persist on the specified unique forecast key.

- [ ] **Step 4: Verify green**

Run: go test ./internal/domain/demand ./internal/service ./internal/repository -run TestCalculate

Expected: PASS.

- [ ] **Step 5: Commit**

~~~text
git add internal/domain/demand internal/service/forecast_service.go internal/repository/forecast_repository.go
git commit -m "feat: calculate and store demand forecasts"
~~~

### Task 5: Explainable price recommendation

**Files:**
- Create: internal/domain/pricing/{optimizer.go,optimizer_test.go}, internal/repository/recommendation_repository.go, internal/service/{recommendation_service.go,recommendation_service_test.go}.

**Produces:** pricing.Optimize(pricing.Input) pricing.Recommendation; stored response includes reason codes and explanation.

- [ ] **Step 1: Write failing guard tests**

~~~go
func TestOptimizeKeepsCurrentPriceDuringPromotion(t *testing.T) {
    got := Optimize(Input{CurrentPrice: decimal.NewFromInt(100), PromotionActive: true})
    require.Equal(t, "promotion_active", got.Status)
    require.True(t, got.RecommendedPrice.Equal(decimal.NewFromInt(100)))
}
func TestOptimizeNeverCrossesMarginFloor(t *testing.T) {
    got := Optimize(inputWithMarginFloor(decimal.NewFromInt(95)))
    require.True(t, got.RecommendedPrice.GreaterThanOrEqual(decimal.NewFromInt(95)))
    require.Contains(t, got.ReasonCodes, "MARGIN_FLOOR_APPLIED")
}
~~~

- [ ] **Step 2: Verify red**

Run: go test ./internal/domain/pricing -run TestOptimize

Expected: FAIL because Optimize is absent.

- [ ] **Step 3: Implement candidates and guards**

First handle promotion and under-14-observation results. Evaluate the five stated candidates, clamp to min/max/margin floor, use elasticity demand and expected gross margin, add the approved 2% stock/trend bias, retain current price below 1% improvement, and write reasoned explanation text.

- [ ] **Step 4: Verify green**

Run: go test ./internal/domain/pricing ./internal/service ./internal/repository -run TestOptimize

Expected: PASS for price bounds, margin floor, low history, promotion and both stock/trend cases.

- [ ] **Step 5: Commit**

~~~text
git add internal/domain/pricing internal/service/recommendation_service.go internal/repository/recommendation_repository.go
git commit -m "feat: add explainable price recommendations"
~~~

### Task 6: API, dashboard, and operations

**Files:**
- Create: internal/transport/http/{router,handlers,errors,handlers_test}.go, internal/app/{config,app,config_test}.go, internal/observability/{metrics,middleware}.go, cmd/api/main.go, web/templates/index.html, web/static/{dashboard.js,styles.css}, docs/api.md.

**Produces:** required API routes, consistent JSON errors, root dashboard, request/metrics middleware, health/readiness, graceful shutdown.

- [ ] **Step 1: Write failing HTTP contract test**

~~~go
func TestDemandUnknownStoreReturnsConsistentNotFound(t *testing.T) {
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/stores/UNKNOWN/products/DRILL-18V/demand", nil))
    require.Equal(t, http.StatusNotFound, rec.Code)
    require.JSONEq(t, "{\"error\":{\"code\":\"not_found\",\"message\":\"store not found\"}}", rec.Body.String())
}
~~~

- [ ] **Step 2: Verify red**

Run: go test ./internal/transport/http -run TestDemandUnknownStoreReturnsConsistentNotFound

Expected: FAIL because router is absent.

- [ ] **Step 3: Implement end-user surface**

Add catalog/pagination, demand/recommendation reads, multipart import, JSON recalculate endpoints, dashboard with selection/cards/28-day chart and promotion/low-confidence flags. Wire safe config defaults, database deadlines, server timeouts, JSON slog, request IDs, recoverer, request metrics, calculation/import/query-failure metrics, and signal shutdown.

- [ ] **Step 4: Verify green**

Run: go test ./internal/transport/http ./internal/app ./internal/observability && go vet ./...

Expected: PASS for happy paths, validation errors, unknown lookup and config defaults.

- [ ] **Step 5: Commit**

~~~text
git add internal/transport internal/app internal/observability cmd/api web docs/api.md
git commit -m "feat: expose observable pricing API and dashboard"
~~~

### Task 7: Delivery, CI, and acceptance verification

**Files:**
- Create: README.md, docs/{architecture,decisions}.md, .github/workflows/ci.yml.
- Modify: deploy/Dockerfile, deploy/docker-compose.yml, Makefile.

**Produces:** one-command startup, hiring-manager README, architecture docs and clean-checkout CI.

- [ ] **Step 1: Write failing container smoke check**

~~~bash
docker compose -f deploy/docker-compose.yml up --build -d
curl --fail http://localhost:8080/healthz
curl --fail http://localhost:8080/api/v1/stores
~~~

- [ ] **Step 2: Verify red**

Run the smoke check before final Compose wiring.

Expected: FAIL until migration/API orchestration is complete.

- [ ] **Step 3: Complete delivery artifacts**

Make Compose wait for PostgreSQL and run migrations; add run, test, lint, seed, ingest, compose-up targets. Write README problem statement, explicit synthetic/deterministic warning, diagram/data flow, startup/curl/API examples, dashboard screenshot, algorithm explanation, tests/CI/trade-offs/role alignment. CI runs tidy-diff, gofmt, vet, PostgreSQL integration tests and Docker build.

- [ ] **Step 4: Verify acceptance**

Run: go mod tidy && go test ./... && go vet ./... && docker compose -f deploy/docker-compose.yml up --build -d

Expected: all tests pass; health, readiness, metrics, dashboard and APIs respond; each seed scenario is visible.

- [ ] **Step 5: Commit and publish**

~~~text
git add README.md docs .github Makefile deploy
git commit -m "docs: complete portfolio delivery and CI"
git push
~~~

## Final verification

- [ ] Re-import the supplied CSV and verify no duplicate daily-sales rows result.
- [ ] Verify every required endpoint and error shape against docs/api.md.
- [ ] Verify the README never claims real retailer data, production ML, or commercially validated pricing.
- [ ] Check GitHub Actions for the pushed branch.


GO ?= go
DATABASE_URL ?= postgres://pricing:pricing@localhost:5432/pricing?sslmode=disable

.PHONY: all tidy fmt test vet run-api seed ingest compose-up compose-down

all: test

tidy:
	$(GO) mod tidy

fmt:
	$(GO)fmt -w $$(find . -name '*.go' -not -path './.git/*')

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

run-api:
	DATABASE_URL=$(DATABASE_URL) $(GO) run ./cmd/api

seed:
	DATABASE_URL=$(DATABASE_URL) $(GO) run ./cmd/seed --load-db

ingest:
	DATABASE_URL=$(DATABASE_URL) $(GO) run ./cmd/ingest --file ./db/seeds/daily_sales.csv

compose-up:
	docker compose -f deploy/docker-compose.yml up --build

compose-down:
	docker compose -f deploy/docker-compose.yml down

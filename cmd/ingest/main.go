package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/KyryloY/demand-pricing-go/internal/repository"
	"github.com/KyryloY/demand-pricing-go/internal/service"
)

func main() {
	filePath := flag.String("file", "./db/seeds/daily_sales.csv", "sales CSV path")
	databaseURL := flag.String("database-url", envOr("DATABASE_URL", "postgres://pricing:pricing@localhost:5432/pricing?sslmode=disable"), "PostgreSQL connection URL")
	flag.Parse()

	file, err := os.Open(*filePath)
	if err != nil {
		fatalf("open sales CSV: %v", err)
	}
	defer file.Close()

	pool, err := pgxpool.New(context.Background(), *databaseURL)
	if err != nil {
		fatalf("create database pool: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(context.Background()); err != nil {
		fatalf("ping database: %v", err)
	}

	importer := service.NewSalesImporter(repository.NewSalesRepository(pool))
	summary, err := importer.Import(context.Background(), file)
	if err != nil {
		fatalf("import sales: %v", err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(summary); err != nil {
		fatalf("write import summary: %v", err)
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

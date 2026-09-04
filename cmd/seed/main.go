package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/KyryloY/demand-pricing-go/internal/app"
	"github.com/KyryloY/demand-pricing-go/internal/seed"
)

func main() {
	outPath := flag.String("csv-out", "./db/seeds/daily_sales.csv", "path for the generated daily sales CSV")
	databaseURL := flag.String("database-url", envOr("DATABASE_URL", ""), "optional PostgreSQL connection URL for catalog loading")
	loadDatabase := flag.Bool("load-db", false, "load stores, products, promotions, and inventory into PostgreSQL")
	skipCSV := flag.Bool("skip-csv", false, "skip writing the generated sales CSV")
	flag.Parse()

	if !*skipCSV {
		if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "create seed directory: %v\n", err)
			os.Exit(1)
		}
		file, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "create seed CSV: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		if err := seed.Generate(file, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)); err != nil {
			fmt.Fprintf(os.Stderr, "generate seed CSV: %v\n", err)
			os.Exit(1)
		}
	}
	if *loadDatabase {
		if *databaseURL == "" {
			fatalf("--load-db requires --database-url or DATABASE_URL")
		}
		pool, err := pgxpool.New(context.Background(), *databaseURL)
		if err != nil {
			fatalf("create database pool: %v", err)
		}
		defer pool.Close()
		if err := pool.Ping(context.Background()); err != nil {
			fatalf("ping database: %v", err)
		}
		if err := app.RunMigrations(*databaseURL, "./db/migrations"); err != nil {
			fatalf("run migrations: %v", err)
		}
		if err := seed.LoadCatalog(context.Background(), pool, time.Date(2025, 4, 30, 0, 0, 0, 0, time.UTC)); err != nil {
			fatalf("load catalog: %v", err)
		}
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

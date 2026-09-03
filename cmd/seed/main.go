package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/KyryloY/demand-pricing-go/internal/seed"
)

func main() {
	outPath := flag.String("csv-out", "./db/seeds/daily_sales.csv", "path for the generated daily sales CSV")
	flag.Parse()

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

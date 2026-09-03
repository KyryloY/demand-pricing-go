package migrations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitialMigrationDefinesCoreTablesAndSalesConstraint(t *testing.T) {
	path := filepath.Join("000001_initial_schema.up.sql")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	for _, table := range []string{
		"stores",
		"products",
		"daily_sales",
		"inventory_snapshots",
		"promotions",
		"demand_forecasts",
		"price_recommendations",
	} {
		if !strings.Contains(string(contents), "CREATE TABLE "+table) {
			t.Errorf("missing table %q", table)
		}
	}

	if !strings.Contains(string(contents), "UNIQUE (store_id, product_id, sale_date)") {
		t.Error("daily_sales must be unique per store, product, and date")
	}
}

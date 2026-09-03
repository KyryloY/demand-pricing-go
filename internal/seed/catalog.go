package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LoadCatalog creates the deterministic stores, products, inventory, and promotions
// consumed by the generated sales CSV.
func LoadCatalog(ctx context.Context, db *pgxpool.Pool, asOf time.Time) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin catalog seed transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for storeIndex := 0; storeIndex < storeCount; storeIndex++ {
		if _, err := tx.Exec(ctx, `
			INSERT INTO stores (code, name, country_code)
			VALUES ($1, $2, 'DE') ON CONFLICT (code) DO NOTHING`,
			fmt.Sprintf("MANNHEIM-%02d", storeIndex+1), fmt.Sprintf("Mannheim Store %02d", storeIndex+1)); err != nil {
			return fmt.Errorf("insert store: %w", err)
		}
	}
	for productIndex := 0; productIndex < productCount; productIndex++ {
		price := 12.0 + float64(productIndex*3)
		if _, err := tx.Exec(ctx, `
			INSERT INTO products (sku, name, category, cost_price, min_margin_pct, min_price, max_price, price_elasticity)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (sku) DO NOTHING`, skuFor(productIndex), "Synthetic "+skuFor(productIndex), categoryFor(productIndex),
			price*0.6, 0.2, price*0.8, price*1.2, -1.0-float64(productIndex%4)*0.1); err != nil {
			return fmt.Errorf("insert product: %w", err)
		}
	}

	for storeIndex := 0; storeIndex < storeCount; storeIndex++ {
		for productIndex := 0; productIndex < productCount; productIndex++ {
			onHand := 100
			switch productIndex {
			case 0:
				onHand = 1000
			case 1:
				onHand = 20
			case 3:
				onHand = 250
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO inventory_snapshots (store_id, product_id, snapshot_date, on_hand_units)
				SELECT s.id, p.id, $3, $4 FROM stores AS s CROSS JOIN products AS p
				WHERE s.code = $1 AND p.sku = $2
				ON CONFLICT (store_id, product_id, snapshot_date) DO UPDATE SET on_hand_units = EXCLUDED.on_hand_units`,
				fmt.Sprintf("MANNHEIM-%02d", storeIndex+1), skuFor(productIndex), asOf, onHand); err != nil {
				return fmt.Errorf("insert inventory snapshot: %w", err)
			}
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO promotions (code, product_id, name, discount_pct, starts_on, ends_on)
		SELECT 'PROMO-PAINT-PRO', id, 'Paint launch promotion', 0.15, $1, $2
		FROM products WHERE sku = 'PAINT-PRO'
		ON CONFLICT (code) DO UPDATE SET ends_on = EXCLUDED.ends_on`, asOf.AddDate(0, 0, -20), asOf.AddDate(0, 0, 20)); err != nil {
		return fmt.Errorf("insert active promotion: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO promotions (code, product_id, name, discount_pct, starts_on, ends_on)
		SELECT 'PROMO-TILE-OLD', id, 'Historical tile promotion', 0.10, $1, $2
		FROM products WHERE sku = 'TILE-CUTTER'
		ON CONFLICT (code) DO UPDATE SET ends_on = EXCLUDED.ends_on`, asOf.AddDate(0, 0, -90), asOf.AddDate(0, 0, -60)); err != nil {
		return fmt.Errorf("insert historical promotion: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit catalog seed transaction: %w", err)
	}
	return nil
}

func categoryFor(productIndex int) string {
	categories := []string{"tools", "tools", "paint", "tile", "accessories"}
	if productIndex < len(categories) {
		return categories[productIndex]
	}
	return "hardware"
}

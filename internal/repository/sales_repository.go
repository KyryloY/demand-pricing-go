package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/KyryloY/demand-pricing-go/internal/domain/sales"
)

type SalesRepository struct {
	db      *pgxpool.Pool
	catalog *CatalogRepository
}

func NewSalesRepository(db *pgxpool.Pool) *SalesRepository {
	return &SalesRepository{db: db, catalog: NewCatalogRepository(db)}
}

func (r *SalesRepository) Resolve(ctx context.Context, storeCode, sku, promotionCode string) (int64, int64, *int64, error) {
	return r.catalog.Resolve(ctx, storeCode, sku, promotionCode)
}

func (r *SalesRepository) Upsert(ctx context.Context, rows []sales.Record) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin sales transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	const query = `
		INSERT INTO daily_sales (
			store_id, product_id, sale_date, units_sold, revenue, unit_price, promotion_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (store_id, product_id, sale_date) DO UPDATE SET
			units_sold = EXCLUDED.units_sold,
			revenue = EXCLUDED.revenue,
			unit_price = EXCLUDED.unit_price,
			promotion_id = EXCLUDED.promotion_id`
	for _, row := range rows {
		if _, err := tx.Exec(ctx, query, row.StoreID, row.ProductID, row.SaleDate, row.UnitsSold, row.Revenue, row.UnitPrice, row.PromotionID); err != nil {
			return 0, fmt.Errorf("upsert sale for %s: %w", row.SaleDate.Format("2006-01-02"), err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit sales transaction: %w", err)
	}
	return len(rows), nil
}

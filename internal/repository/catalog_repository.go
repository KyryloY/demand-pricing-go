package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CatalogRepository struct {
	db *pgxpool.Pool
}

func NewCatalogRepository(db *pgxpool.Pool) *CatalogRepository {
	return &CatalogRepository{db: db}
}

func (r *CatalogRepository) Resolve(ctx context.Context, storeCode, sku, promotionCode string) (storeID, productID int64, promotionID *int64, err error) {
	if promotionCode == "" {
		err = r.db.QueryRow(ctx, `
			SELECT s.id, p.id
			FROM stores AS s
			CROSS JOIN products AS p
			WHERE s.code = $1 AND p.sku = $2`, storeCode, sku).Scan(&storeID, &productID)
		if err != nil {
			if err == pgx.ErrNoRows {
				return 0, 0, nil, fmt.Errorf("store or product not found")
			}
			return 0, 0, nil, fmt.Errorf("resolve store and product: %w", err)
		}
		return storeID, productID, nil, nil
	}

	err = r.db.QueryRow(ctx, `
		SELECT s.id, p.id, pr.id
		FROM stores AS s
		CROSS JOIN products AS p
		JOIN promotions AS pr
		  ON pr.code = $3
		 AND pr.product_id = p.id
		 AND (pr.store_id IS NULL OR pr.store_id = s.id)
		WHERE s.code = $1 AND p.sku = $2`, storeCode, sku, promotionCode).Scan(&storeID, &productID, &promotionID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, 0, nil, fmt.Errorf("store, product, or promotion not found")
		}
		return 0, 0, nil, fmt.Errorf("resolve store, product, and promotion: %w", err)
	}
	return storeID, productID, promotionID, nil
}

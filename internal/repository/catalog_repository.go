package repository

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/KyryloY/demand-pricing-go/internal/domain/catalog"
)

type CatalogRepository struct {
	db *pgxpool.Pool
}

func (r *CatalogRepository) ListStores(ctx context.Context) ([]catalog.Store, error) {
	rows, err := r.db.Query(ctx, `SELECT id, code, name, country_code, created_at FROM stores ORDER BY code`)
	if err != nil {
		return nil, fmt.Errorf("query stores: %w", err)
	}
	defer rows.Close()
	stores := make([]catalog.Store, 0)
	for rows.Next() {
		var store catalog.Store
		if err := rows.Scan(&store.ID, &store.Code, &store.Name, &store.CountryCode, &store.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan store: %w", err)
		}
		stores = append(stores, store)
	}
	return stores, rows.Err()
}

func (r *CatalogRepository) ListProducts(ctx context.Context, search, category string, limit, offset int) ([]catalog.Product, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, sku, name, category, cost_price, min_margin_pct, min_price, max_price, price_elasticity
		FROM products
		WHERE ($1 = '' OR sku ILIKE '%' || $1 || '%' OR name ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR category = $2)
		ORDER BY sku LIMIT $3 OFFSET $4`, search, category, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query products: %w", err)
	}
	defer rows.Close()
	products := make([]catalog.Product, 0)
	for rows.Next() {
		product, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	return products, rows.Err()
}

func (r *CatalogRepository) Product(ctx context.Context, sku string) (catalog.Product, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, sku, name, category, cost_price, min_margin_pct, min_price, max_price, price_elasticity
		FROM products WHERE sku = $1`, sku)
	product, err := scanProduct(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return catalog.Product{}, fmt.Errorf("product not found")
		}
		return catalog.Product{}, fmt.Errorf("query product: %w", err)
	}
	return product, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanProduct(row rowScanner) (catalog.Product, error) {
	var product catalog.Product
	var cost, margin, minPrice, maxPrice, elasticity float64
	err := row.Scan(&product.ID, &product.SKU, &product.Name, &product.Category, &cost, &margin, &minPrice, &maxPrice, &elasticity)
	if err != nil {
		return catalog.Product{}, err
	}
	product.CostPrice = strconv.FormatFloat(cost, 'f', 2, 64)
	product.MinMarginPct = strconv.FormatFloat(margin, 'f', 4, 64)
	product.MinPrice = strconv.FormatFloat(minPrice, 'f', 2, 64)
	product.MaxPrice = strconv.FormatFloat(maxPrice, 'f', 2, 64)
	product.PriceElasticity = strconv.FormatFloat(elasticity, 'f', 4, 64)
	return product, nil
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

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/KyryloY/demand-pricing-go/internal/domain/pricing"
)

type RecommendationRepository struct {
	db *pgxpool.Pool
}

func NewRecommendationRepository(db *pgxpool.Pool) *RecommendationRepository {
	return &RecommendationRepository{db: db}
}

func (r *RecommendationRepository) LoadRecommendationInput(ctx context.Context, storeCode, sku string, recommendationDate time.Time) (pricing.Input, error) {
	var input pricing.Input
	var confidence float64
	err := r.db.QueryRow(ctx, `
		SELECT
			p.cost_price, p.min_margin_pct, p.min_price, p.max_price, p.price_elasticity,
			COALESCE((SELECT ds.unit_price FROM daily_sales AS ds
				WHERE ds.store_id = s.id AND ds.product_id = p.id
				ORDER BY ds.sale_date DESC LIMIT 1), p.min_price),
			COALESCE((SELECT df.forecast_daily_units FROM demand_forecasts AS df
				WHERE df.store_id = s.id AND df.product_id = p.id AND df.forecast_date <= $3
				ORDER BY df.forecast_date DESC LIMIT 1), 0),
			COALESCE((SELECT df.trend_pct FROM demand_forecasts AS df
				WHERE df.store_id = s.id AND df.product_id = p.id AND df.forecast_date <= $3
				ORDER BY df.forecast_date DESC LIMIT 1), 0),
			COALESCE((SELECT df.confidence FROM demand_forecasts AS df
				WHERE df.store_id = s.id AND df.product_id = p.id AND df.forecast_date <= $3
				ORDER BY df.forecast_date DESC LIMIT 1), 0),
			COALESCE((SELECT inv.on_hand_units FROM inventory_snapshots AS inv
				WHERE inv.store_id = s.id AND inv.product_id = p.id AND inv.snapshot_date <= $3
				ORDER BY inv.snapshot_date DESC LIMIT 1), 0),
			EXISTS (SELECT 1 FROM promotions AS pr
				WHERE pr.product_id = p.id AND (pr.store_id IS NULL OR pr.store_id = s.id)
				  AND pr.starts_on <= $3 AND pr.ends_on >= $3)
		FROM stores AS s CROSS JOIN products AS p
		WHERE s.code = $1 AND p.sku = $2`, storeCode, sku, recommendationDate).Scan(
		&input.CostPrice, &input.MinMarginPct, &input.MinPrice, &input.MaxPrice, &input.PriceElasticity,
		&input.CurrentPrice, &input.BaselineDemand, &input.TrendPct, &confidence, &input.InventoryUnits, &input.PromotionActive)
	if err != nil {
		if err == pgx.ErrNoRows {
			return pricing.Input{}, fmt.Errorf("store or product not found")
		}
		return pricing.Input{}, fmt.Errorf("query recommendation input: %w", err)
	}
	input.ObservationCount = int(math.Round(confidence * 28))
	return input, nil
}

func (r *RecommendationRepository) UpsertRecommendation(ctx context.Context, storeCode, sku string, recommendationDate time.Time, input pricing.Input, recommendation pricing.Recommendation) error {
	reasonCodes, err := json.Marshal(recommendation.ReasonCodes)
	if err != nil {
		return fmt.Errorf("encode recommendation reasons: %w", err)
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO price_recommendations (
			store_id, product_id, recommendation_date, current_price, recommended_price,
			expected_daily_units, expected_daily_margin, status, reason_codes
		)
		SELECT s.id, p.id, $3, $4, $5, $6, $7, $8, $9
		FROM stores AS s CROSS JOIN products AS p
		WHERE s.code = $1 AND p.sku = $2
		ON CONFLICT (store_id, product_id, recommendation_date) DO UPDATE SET
			current_price = EXCLUDED.current_price,
			recommended_price = EXCLUDED.recommended_price,
			expected_daily_units = EXCLUDED.expected_daily_units,
			expected_daily_margin = EXCLUDED.expected_daily_margin,
			status = EXCLUDED.status,
			reason_codes = EXCLUDED.reason_codes,
			created_at = CURRENT_TIMESTAMP`, storeCode, sku, recommendationDate, input.CurrentPrice,
		recommendation.RecommendedPrice, recommendation.ExpectedDailyUnits, recommendation.ExpectedDailyMargin,
		recommendation.Status, reasonCodes)
	if err != nil {
		return fmt.Errorf("upsert price recommendation: %w", err)
	}
	return nil
}

package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/KyryloY/demand-pricing-go/internal/domain/demand"
)

type ForecastRepository struct {
	db *pgxpool.Pool
}

func NewForecastRepository(db *pgxpool.Pool) *ForecastRepository {
	return &ForecastRepository{db: db}
}

func (r *ForecastRepository) ResolveForecastTarget(ctx context.Context, storeCode, sku string) (int64, int64, error) {
	var storeID, productID int64
	err := r.db.QueryRow(ctx, `
		SELECT s.id, p.id
		FROM stores AS s CROSS JOIN products AS p
		WHERE s.code = $1 AND p.sku = $2`, storeCode, sku).Scan(&storeID, &productID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, 0, fmt.Errorf("store or product not found")
		}
		return 0, 0, fmt.Errorf("resolve forecast target: %w", err)
	}
	return storeID, productID, nil
}

func (r *ForecastRepository) Observations(ctx context.Context, storeID, productID int64, asOf time.Time) ([]demand.Observation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT sale_date, units_sold, promotion_id IS NOT NULL
		FROM daily_sales
		WHERE store_id = $1
		  AND product_id = $2
		  AND sale_date BETWEEN $3::date - 27 AND $3::date
		ORDER BY sale_date`, storeID, productID, asOf)
	if err != nil {
		return nil, fmt.Errorf("query sales observations: %w", err)
	}
	defer rows.Close()

	observations := make([]demand.Observation, 0)
	for rows.Next() {
		var observation demand.Observation
		if err := rows.Scan(&observation.Date, &observation.UnitsSold, &observation.PromotionActive); err != nil {
			return nil, fmt.Errorf("scan sales observation: %w", err)
		}
		observations = append(observations, observation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sales observations: %w", err)
	}
	return observations, nil
}

func (r *ForecastRepository) UpsertForecast(ctx context.Context, storeID, productID int64, forecastDate time.Time, forecast demand.Forecast) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO demand_forecasts (
			store_id, product_id, forecast_date, forecast_daily_units, trend_pct, confidence
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (store_id, product_id, forecast_date) DO UPDATE SET
			forecast_daily_units = EXCLUDED.forecast_daily_units,
			trend_pct = EXCLUDED.trend_pct,
			confidence = EXCLUDED.confidence,
			created_at = CURRENT_TIMESTAMP`, storeID, productID, forecastDate, forecast.DailyUnits, forecast.TrendPct, forecast.Confidence)
	if err != nil {
		return fmt.Errorf("upsert demand forecast: %w", err)
	}
	return nil
}

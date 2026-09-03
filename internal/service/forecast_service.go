package service

import (
	"context"
	"fmt"
	"time"

	"github.com/KyryloY/demand-pricing-go/internal/domain/demand"
)

type ForecastRepository interface {
	ResolveForecastTarget(ctx context.Context, storeCode, sku string) (storeID, productID int64, err error)
	Observations(ctx context.Context, storeID, productID int64, asOf time.Time) ([]demand.Observation, error)
	UpsertForecast(ctx context.Context, storeID, productID int64, forecastDate time.Time, forecast demand.Forecast) error
}

type ForecastService struct {
	repository ForecastRepository
}

func NewForecastService(repository ForecastRepository) *ForecastService {
	return &ForecastService{repository: repository}
}

func (s *ForecastService) Recalculate(ctx context.Context, storeCode, sku string, asOf time.Time) (demand.Forecast, error) {
	if s == nil || s.repository == nil {
		return demand.Forecast{}, fmt.Errorf("forecast repository is required")
	}
	storeID, productID, err := s.repository.ResolveForecastTarget(ctx, storeCode, sku)
	if err != nil {
		return demand.Forecast{}, fmt.Errorf("resolve forecast target: %w", err)
	}
	observations, err := s.repository.Observations(ctx, storeID, productID, asOf)
	if err != nil {
		return demand.Forecast{}, fmt.Errorf("load forecast observations: %w", err)
	}
	forecast, err := demand.Calculate(asOf, observations)
	if err != nil {
		return demand.Forecast{}, fmt.Errorf("calculate forecast: %w", err)
	}
	if err := s.repository.UpsertForecast(ctx, storeID, productID, asOf, forecast); err != nil {
		return demand.Forecast{}, fmt.Errorf("store forecast: %w", err)
	}
	return forecast, nil
}

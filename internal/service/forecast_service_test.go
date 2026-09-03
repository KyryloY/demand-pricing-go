package service

import (
	"context"
	"testing"
	"time"

	"github.com/KyryloY/demand-pricing-go/internal/domain/demand"
)

type forecastTestRepository struct {
	observations []demand.Observation
	saved        demand.Forecast
}

func (r *forecastTestRepository) ResolveForecastTarget(context.Context, string, string) (int64, int64, error) {
	return 7, 11, nil
}

func (r *forecastTestRepository) Observations(context.Context, int64, int64, time.Time) ([]demand.Observation, error) {
	return r.observations, nil
}

func (r *forecastTestRepository) UpsertForecast(_ context.Context, _ int64, _ int64, _ time.Time, forecast demand.Forecast) error {
	r.saved = forecast
	return nil
}

func TestForecastServiceRecalculatePersistsCalculatedForecast(t *testing.T) {
	asOf := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)
	observations := make([]demand.Observation, 0, 28)
	for day := 0; day < 28; day++ {
		units := 20
		if day >= 21 {
			units = 10
		}
		observations = append(observations, demand.Observation{
			Date:      asOf.AddDate(0, 0, day-27),
			UnitsSold: units,
		})
	}
	repository := &forecastTestRepository{observations: observations}

	forecast, err := NewForecastService(repository).Recalculate(context.Background(), "MANNHEIM-01", "DRILL-18V", asOf)
	if err != nil {
		t.Fatalf("recalculate: %v", err)
	}
	if forecast.DailyUnits != 13 || repository.saved.DailyUnits != 13 {
		t.Fatalf("forecast = %#v, saved = %#v, want daily units 13", forecast, repository.saved)
	}
}

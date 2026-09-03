package service

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/KyryloY/demand-pricing-go/internal/domain/pricing"
)

type recommendationTestRepository struct {
	input pricing.Input
	saved pricing.Recommendation
}

func (r *recommendationTestRepository) LoadRecommendationInput(context.Context, string, string, time.Time) (pricing.Input, error) {
	return r.input, nil
}

func (r *recommendationTestRepository) UpsertRecommendation(_ context.Context, _ string, _ string, _ time.Time, _ pricing.Input, recommendation pricing.Recommendation) error {
	r.saved = recommendation
	return nil
}

func TestRecommendationServiceRecalculatePersistsPromotionDecision(t *testing.T) {
	repository := &recommendationTestRepository{input: pricing.Input{
		CurrentPrice:     decimal.NewFromInt(100),
		CostPrice:        decimal.NewFromInt(60),
		MinMarginPct:     decimal.NewFromFloat(0.2),
		MinPrice:         decimal.NewFromInt(50),
		MaxPrice:         decimal.NewFromInt(150),
		BaselineDemand:   10,
		ObservationCount: 28,
		PriceElasticity:  -1.2,
		PromotionActive:  true,
	}}

	recommendation, err := NewRecommendationService(repository).Recalculate(context.Background(), "MANNHEIM-01", "PAINT-PRO", time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("recalculate: %v", err)
	}
	if recommendation.Status != "promotion_active" || repository.saved.Status != "promotion_active" {
		t.Fatalf("recommendation = %#v, saved = %#v, want promotion_active", recommendation, repository.saved)
	}
}

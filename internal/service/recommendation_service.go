package service

import (
	"context"
	"fmt"
	"time"

	"github.com/KyryloY/demand-pricing-go/internal/domain/pricing"
)

type RecommendationRepository interface {
	LoadRecommendationInput(ctx context.Context, storeCode, sku string, recommendationDate time.Time) (pricing.Input, error)
	UpsertRecommendation(ctx context.Context, storeCode, sku string, recommendationDate time.Time, input pricing.Input, recommendation pricing.Recommendation) error
}

type RecommendationService struct {
	repository RecommendationRepository
}

func NewRecommendationService(repository RecommendationRepository) *RecommendationService {
	return &RecommendationService{repository: repository}
}

func (s *RecommendationService) Recalculate(ctx context.Context, storeCode, sku string, recommendationDate time.Time) (pricing.Recommendation, error) {
	if s == nil || s.repository == nil {
		return pricing.Recommendation{}, fmt.Errorf("recommendation repository is required")
	}
	input, err := s.repository.LoadRecommendationInput(ctx, storeCode, sku, recommendationDate)
	if err != nil {
		return pricing.Recommendation{}, fmt.Errorf("load recommendation input: %w", err)
	}
	recommendation := pricing.Optimize(input)
	if err := s.repository.UpsertRecommendation(ctx, storeCode, sku, recommendationDate, input, recommendation); err != nil {
		return pricing.Recommendation{}, fmt.Errorf("store recommendation: %w", err)
	}
	return recommendation, nil
}

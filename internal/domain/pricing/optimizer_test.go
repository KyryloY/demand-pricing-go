package pricing

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestOptimizeKeepsCurrentPriceDuringPromotion(t *testing.T) {
	current := decimal.NewFromInt(100)
	got := Optimize(Input{
		CurrentPrice:     current,
		CostPrice:        decimal.NewFromInt(60),
		MinMarginPct:     decimal.NewFromFloat(0.2),
		MinPrice:         decimal.NewFromInt(50),
		MaxPrice:         decimal.NewFromInt(150),
		BaselineDemand:   10,
		ObservationCount: 28,
		PriceElasticity:  -1.2,
		PromotionActive:  true,
	})
	if got.Status != "promotion_active" {
		t.Fatalf("status = %q, want promotion_active", got.Status)
	}
	if !got.RecommendedPrice.Equal(current) {
		t.Fatalf("recommended price = %s, want %s", got.RecommendedPrice, current)
	}
}

func TestOptimizeNeverCrossesMarginFloor(t *testing.T) {
	got := Optimize(Input{
		CurrentPrice:     decimal.NewFromInt(100),
		CostPrice:        decimal.NewFromInt(90),
		MinMarginPct:     decimal.NewFromFloat(0.1),
		MinPrice:         decimal.NewFromInt(1),
		MaxPrice:         decimal.NewFromInt(150),
		BaselineDemand:   10,
		ObservationCount: 28,
		PriceElasticity:  -1.2,
	})
	marginFloor := decimal.NewFromInt(99)
	if got.RecommendedPrice.LessThan(marginFloor) {
		t.Fatalf("recommended price = %s, must be at least %s", got.RecommendedPrice, marginFloor)
	}
	if !containsReason(got.ReasonCodes, "MARGIN_FLOOR_APPLIED") {
		t.Fatalf("reason codes = %#v, want MARGIN_FLOOR_APPLIED", got.ReasonCodes)
	}
}

func containsReason(reasons []string, want string) bool {
	for _, reason := range reasons {
		if reason == want {
			return true
		}
	}
	return false
}

package pricing

import (
	"fmt"
	"math"

	"github.com/shopspring/decimal"
)

type Input struct {
	CurrentPrice     decimal.Decimal
	CostPrice        decimal.Decimal
	MinMarginPct     decimal.Decimal
	MinPrice         decimal.Decimal
	MaxPrice         decimal.Decimal
	BaselineDemand   float64
	InventoryUnits   float64
	TrendPct         float64
	PriceElasticity  float64
	PromotionActive  bool
	ObservationCount int
}

type Recommendation struct {
	RecommendedPrice    decimal.Decimal
	ExpectedDailyUnits  float64
	ExpectedDailyMargin decimal.Decimal
	Status              string
	ReasonCodes         []string
	Explanation         string
}

func Optimize(input Input) Recommendation {
	marginFloor := input.CostPrice.Mul(decimal.NewFromInt(1).Add(input.MinMarginPct))
	permittedFloor := input.MinPrice
	if marginFloor.GreaterThan(permittedFloor) {
		permittedFloor = marginFloor
	}
	reasons := make([]string, 0, 4)
	if marginFloor.GreaterThan(input.MinPrice) {
		reasons = append(reasons, "MARGIN_FLOOR_APPLIED")
	}
	if input.CurrentPrice.IsNegative() || input.MaxPrice.LessThan(permittedFloor) {
		return Recommendation{
			RecommendedPrice: permittedFloor,
			Status:           "invalid_input",
			ReasonCodes:      reasons,
			Explanation:      "price bounds cannot satisfy the configured margin floor",
		}
	}

	currentPrice := clamp(input.CurrentPrice, permittedFloor, input.MaxPrice)
	currentUnits := estimatedDemand(input.BaselineDemand, currentPrice, currentPrice, input.PriceElasticity)
	currentMargin := grossMargin(currentUnits, currentPrice, input.CostPrice)
	if input.PromotionActive {
		reasons = append(reasons, "PROMOTION_ACTIVE")
		return Recommendation{
			RecommendedPrice:    currentPrice,
			ExpectedDailyUnits:  currentUnits,
			ExpectedDailyMargin: currentMargin,
			Status:              "promotion_active",
			ReasonCodes:         reasons,
			Explanation:         "an active promotion keeps the current price unchanged",
		}
	}
	if input.ObservationCount < 14 {
		reasons = append(reasons, "LOW_CONFIDENCE")
		return Recommendation{
			RecommendedPrice:    currentPrice,
			ExpectedDailyUnits:  currentUnits,
			ExpectedDailyMargin: currentMargin,
			Status:              "low_confidence",
			ReasonCodes:         reasons,
			Explanation:         "fewer than 14 valid observations are available for a price change",
		}
	}

	stockCover := input.InventoryUnits / math.Max(input.BaselineDemand, 0.1)
	favorLower := stockCover > 21 && input.TrendPct < -0.10
	favorHigher := stockCover < 7 && input.TrendPct > 0.10
	if favorLower {
		reasons = append(reasons, "HIGH_STOCK_COVER", "NEGATIVE_DEMAND_TREND")
	}
	if favorHigher {
		reasons = append(reasons, "LOW_STOCK_COVER", "POSITIVE_DEMAND_TREND")
	}

	type candidateResult struct {
		price  decimal.Decimal
		units  float64
		margin decimal.Decimal
		score  float64
	}
	best := candidateResult{price: currentPrice, units: currentUnits, margin: currentMargin, score: currentMargin.InexactFloat64()}
	for _, multiplier := range []float64{0.90, 0.95, 1.00, 1.03, 1.05} {
		price := clamp(input.CurrentPrice.Mul(decimal.NewFromFloat(multiplier)).Round(2), permittedFloor, input.MaxPrice)
		units := estimatedDemand(input.BaselineDemand, price, currentPrice, input.PriceElasticity)
		margin := grossMargin(units, price, input.CostPrice)
		score := margin.InexactFloat64()
		if (favorLower && price.LessThan(currentPrice)) || (favorHigher && price.GreaterThan(currentPrice)) {
			score *= 1.02
		}
		if score > best.score {
			best = candidateResult{price: price, units: units, margin: margin, score: score}
		}
	}

	if best.margin.LessThanOrEqual(currentMargin.Mul(decimal.NewFromFloat(1.01))) {
		return Recommendation{
			RecommendedPrice:    currentPrice,
			ExpectedDailyUnits:  currentUnits,
			ExpectedDailyMargin: currentMargin,
			Status:              "hold",
			ReasonCodes:         reasons,
			Explanation:         "the expected gross-margin improvement is below the 1% decision threshold",
		}
	}
	return Recommendation{
		RecommendedPrice:    best.price,
		ExpectedDailyUnits:  best.units,
		ExpectedDailyMargin: best.margin,
		Status:              "recommended",
		ReasonCodes:         reasons,
		Explanation:         fmt.Sprintf("candidate price %s has the highest expected daily gross margin", best.price),
	}
}

func clamp(value, lower, upper decimal.Decimal) decimal.Decimal {
	if value.LessThan(lower) {
		return lower
	}
	if value.GreaterThan(upper) {
		return upper
	}
	return value
}

func estimatedDemand(baseline float64, candidate, current decimal.Decimal, elasticity float64) float64 {
	if baseline < 0 || current.IsZero() {
		return 0
	}
	ratio := candidate.Div(current).InexactFloat64()
	return baseline * math.Pow(ratio, elasticity)
}

func grossMargin(units float64, price, cost decimal.Decimal) decimal.Decimal {
	return decimal.NewFromFloat(units).Mul(price.Sub(cost)).Round(2)
}

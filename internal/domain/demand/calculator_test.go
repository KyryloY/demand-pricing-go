package demand

import (
	"testing"
	"time"
)

func TestCalculateUsesWeightedSevenAndTwentyEightDayAverages(t *testing.T) {
	asOf := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)
	observations := make([]Observation, 0, 28)
	for day := 0; day < 28; day++ {
		units := 20
		if day >= 21 {
			units = 10
		}
		observations = append(observations, Observation{
			Date:      asOf.AddDate(0, 0, day-27),
			UnitsSold: units,
		})
	}

	forecast, err := Calculate(asOf, observations)
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	if forecast.DailyUnits != 13 {
		t.Fatalf("daily units = %.4f, want 13", forecast.DailyUnits)
	}
}

func TestCalculateConfidenceUsesNonPromotionObservationCount(t *testing.T) {
	asOf := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)
	observations := make([]Observation, 0, 14)
	for day := 0; day < 14; day++ {
		observations = append(observations, Observation{
			Date:      asOf.AddDate(0, 0, day-13),
			UnitsSold: 12,
		})
	}

	forecast, err := Calculate(asOf, observations)
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	if forecast.Confidence != 0.5 {
		t.Fatalf("confidence = %.4f, want 0.5", forecast.Confidence)
	}
}

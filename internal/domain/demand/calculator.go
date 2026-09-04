package demand

import (
	"fmt"
	"time"
)

type Observation struct {
	Date            time.Time
	UnitsSold       int
	PromotionActive bool
}

type Forecast struct {
	DailyUnits       float64
	TrendPct         float64
	Confidence       float64
	ObservationCount int
}

func Calculate(asOf time.Time, observations []Observation) (Forecast, error) {
	if len(observations) == 0 {
		return Forecast{}, fmt.Errorf("at least one observation is required")
	}
	windowStart := asOf.AddDate(0, 0, -27)
	valid := make([]Observation, 0, len(observations))
	nonPromotion := make([]Observation, 0, len(observations))
	for _, observation := range observations {
		if observation.Date.After(asOf) || observation.Date.Before(windowStart) || observation.UnitsSold < 0 {
			continue
		}
		valid = append(valid, observation)
		if !observation.PromotionActive {
			nonPromotion = append(nonPromotion, observation)
		}
	}
	if len(valid) == 0 {
		return Forecast{}, fmt.Errorf("no valid observations in the requested window")
	}

	baselineObservations := nonPromotion
	if len(nonPromotion) < 14 {
		baselineObservations = valid
	}
	avg7 := averageInWindow(baselineObservations, asOf.AddDate(0, 0, -6))
	avg28 := averageInWindow(baselineObservations, windowStart)
	if avg28 == 0 {
		return Forecast{
			DailyUnits:       0.6 * avg7,
			TrendPct:         0,
			Confidence:       confidence(len(nonPromotion)),
			ObservationCount: len(nonPromotion),
		}, nil
	}
	return Forecast{
		DailyUnits:       0.6*avg7 + 0.4*avg28,
		TrendPct:         (avg7 - avg28) / avg28,
		Confidence:       confidence(len(nonPromotion)),
		ObservationCount: len(nonPromotion),
	}, nil
}

func averageInWindow(observations []Observation, start time.Time) float64 {
	var total float64
	count := 0
	for _, observation := range observations {
		if observation.Date.Before(start) {
			continue
		}
		total += float64(observation.UnitsSold)
		count++
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func confidence(observationCount int) float64 {
	value := float64(observationCount) / 28
	if value > 1 {
		return 1
	}
	return value
}

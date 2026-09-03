package seed

import (
	"encoding/csv"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"time"
)

const (
	storeCount       = 5
	productCount     = 50
	historyDays      = 120
	insufficientDays = 10
)

var scenarioSKUs = [...]string{
	"DRILL-18V",
	"SAW-CORDLESS",
	"PAINT-PRO",
	"TILE-CUTTER",
	"LADDER-ALU",
}

// Generate writes deterministic sales fixtures using startDate as the first day.
// The CSV intentionally contains only the fields accepted by the sales importer.
func Generate(w io.Writer, startDate time.Time) error {
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"store_code", "sku", "sale_date", "units_sold", "unit_price", "promotion_code"}); err != nil {
		return fmt.Errorf("write CSV header: %w", err)
	}

	rng := rand.New(rand.NewSource(20250902))
	for storeIndex := 0; storeIndex < storeCount; storeIndex++ {
		storeCode := fmt.Sprintf("MANNHEIM-%02d", storeIndex+1)
		for productIndex := 0; productIndex < productCount; productIndex++ {
			sku := skuFor(productIndex)
			days := historyDays
			if productIndex == 4 {
				days = insufficientDays
			}

			for dayIndex := 0; dayIndex < days; dayIndex++ {
				date := startDate.UTC().AddDate(0, 0, dayIndex).Format("2006-01-02")
				units := seededUnits(rng, storeIndex, productIndex, dayIndex)
				price := 12.0 + float64(productIndex*3) + float64(storeIndex)*0.5
				promotionCode := ""
				if productIndex == 2 && dayIndex >= 100 {
					promotionCode = "PROMO-PAINT-PRO"
					price *= 0.85
				}

				row := []string{
					storeCode,
					sku,
					date,
					strconv.Itoa(units),
					strconv.FormatFloat(price, 'f', 2, 64),
					promotionCode,
				}
				if err := writer.Write(row); err != nil {
					return fmt.Errorf("write sales row: %w", err)
				}
			}
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush sales CSV: %w", err)
	}
	return nil
}

func skuFor(productIndex int) string {
	if productIndex < len(scenarioSKUs) {
		return scenarioSKUs[productIndex]
	}
	return fmt.Sprintf("DIY-%03d", productIndex+1)
}

func seededUnits(rng *rand.Rand, storeIndex, productIndex, dayIndex int) int {
	base := 18 + productIndex%9 + storeIndex*2
	trend := 0
	switch productIndex {
	case 0:
		trend = -dayIndex / 12
	case 1:
		trend = dayIndex / 12
	case 2:
		trend = dayIndex / 30
	case 3:
		trend = -dayIndex / 20
	}
	noise := rng.Intn(5) - 2
	if units := base + trend + noise; units > 0 {
		return units
	}
	return 1
}

package sales

import (
	"time"

	"github.com/shopspring/decimal"
)

// Record is a resolved daily sales row ready for persistence.
type Record struct {
	StoreID     int64
	ProductID   int64
	PromotionID *int64
	SaleDate    time.Time
	UnitsSold   int
	UnitPrice   decimal.Decimal
	Revenue     decimal.Decimal
}

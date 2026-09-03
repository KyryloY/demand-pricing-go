package catalog

import "time"

type Store struct {
	ID          int64     `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	CountryCode string    `json:"country_code"`
	CreatedAt   time.Time `json:"created_at"`
}

type Product struct {
	ID              int64  `json:"id"`
	SKU             string `json:"sku"`
	Name            string `json:"name"`
	Category        string `json:"category"`
	CostPrice       string `json:"cost_price"`
	MinMarginPct    string `json:"min_margin_pct"`
	MinPrice        string `json:"min_price"`
	MaxPrice        string `json:"max_price"`
	PriceElasticity string `json:"price_elasticity"`
}

package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	domainsales "github.com/KyryloY/demand-pricing-go/internal/domain/sales"
)

// SalesRecord is kept as a service-level alias for the importer port.
type SalesRecord = domainsales.Record

type SalesImportRepository interface {
	Resolve(ctx context.Context, storeCode, sku, promotionCode string) (storeID, productID int64, promotionID *int64, err error)
	Upsert(ctx context.Context, rows []domainsales.Record) (int, error)
}

type SalesImporter struct {
	repository SalesImportRepository
}

type ImportSummary struct {
	Accepted int
	Rejected int
	Upserted int
	Errors   []ImportRowError
}

type ImportRowError struct {
	Row    int
	Reason string
}

func NewSalesImporter(repository SalesImportRepository) *SalesImporter {
	return &SalesImporter{repository: repository}
}

func (i *SalesImporter) Import(ctx context.Context, input io.Reader) (ImportSummary, error) {
	if i == nil || i.repository == nil {
		return ImportSummary{}, fmt.Errorf("sales importer repository is required")
	}

	reader := csv.NewReader(input)
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err == io.EOF {
		return ImportSummary{}, fmt.Errorf("CSV is empty")
	}
	if err != nil {
		return ImportSummary{}, fmt.Errorf("read CSV header: %w", err)
	}
	wantHeader := []string{"store_code", "sku", "sale_date", "units_sold", "unit_price", "promotion_code"}
	if !equalStrings(header, wantHeader) {
		return ImportSummary{}, fmt.Errorf("invalid CSV header: want %s", strings.Join(wantHeader, ","))
	}

	var summary ImportSummary
	rows := make([]domainsales.Record, 0)
	rowNumber := 1
	for {
		rowNumber++
		row, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			summary.Rejected++
			summary.Errors = append(summary.Errors, ImportRowError{Row: rowNumber, Reason: readErr.Error()})
			continue
		}
		record, parseErr := i.parseRow(ctx, row)
		if parseErr != nil {
			summary.Rejected++
			summary.Errors = append(summary.Errors, ImportRowError{Row: rowNumber, Reason: parseErr.Error()})
			continue
		}
		summary.Accepted++
		rows = append(rows, record)
	}

	if len(rows) > 0 {
		summary.Upserted, err = i.repository.Upsert(ctx, rows)
		if err != nil {
			return summary, fmt.Errorf("upsert sales: %w", err)
		}
	}
	return summary, nil
}

func (i *SalesImporter) parseRow(ctx context.Context, row []string) (domainsales.Record, error) {
	if len(row) != 6 {
		return domainsales.Record{}, fmt.Errorf("want 6 fields, got %d", len(row))
	}
	storeCode := strings.TrimSpace(row[0])
	sku := strings.TrimSpace(row[1])
	if storeCode == "" || sku == "" {
		return domainsales.Record{}, fmt.Errorf("store_code and sku are required")
	}
	saleDate, err := time.Parse("2006-01-02", strings.TrimSpace(row[2]))
	if err != nil {
		return domainsales.Record{}, fmt.Errorf("invalid sale_date: %w", err)
	}
	units, err := strconv.Atoi(strings.TrimSpace(row[3]))
	if err != nil || units < 0 {
		return domainsales.Record{}, fmt.Errorf("invalid units_sold")
	}
	unitPrice, err := decimal.NewFromString(strings.TrimSpace(row[4]))
	if err != nil || unitPrice.IsNegative() {
		return domainsales.Record{}, fmt.Errorf("invalid unit_price")
	}
	storeID, productID, promotionID, err := i.repository.Resolve(ctx, storeCode, sku, strings.TrimSpace(row[5]))
	if err != nil {
		return domainsales.Record{}, err
	}
	return domainsales.Record{
		StoreID:     storeID,
		ProductID:   productID,
		PromotionID: promotionID,
		SaleDate:    saleDate,
		UnitsSold:   units,
		UnitPrice:   unitPrice,
		Revenue:     unitPrice.Mul(decimal.NewFromInt(int64(units))),
	}, nil
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

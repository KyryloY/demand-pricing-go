package service

import (
	"context"
	"strings"
	"testing"
)

type importerTestRepository struct{}

func (importerTestRepository) Resolve(context.Context, string, string, string) (int64, int64, *int64, error) {
	return 1, 1, nil, nil
}

func (importerTestRepository) Upsert(context.Context, []SalesRecord) (int, error) {
	return 1, nil
}

func TestImportKeepsValidRowsWhenOneRowIsInvalid(t *testing.T) {
	input := strings.NewReader("store_code,sku,sale_date,units_sold,unit_price,promotion_code\n" +
		"MANNHEIM-01,DRILL-18V,2025-01-01,4,12.00,\n" +
		"MANNHEIM-01,DRILL-18V,not-a-date,5,12.00,\n")

	importer := NewSalesImporter(importerTestRepository{})
	summary, err := importer.Import(context.Background(), input)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if summary.Accepted != 1 {
		t.Fatalf("accepted = %d, want 1", summary.Accepted)
	}
	if summary.Rejected != 1 {
		t.Fatalf("rejected = %d, want 1", summary.Rejected)
	}
	if len(summary.Errors) != 1 || summary.Errors[0].Row != 3 {
		t.Fatalf("errors = %#v, want one error on row 3", summary.Errors)
	}
}

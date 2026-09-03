package seed

import (
	"bytes"
	"testing"
	"time"
)

func TestGenerateIsReproducible(t *testing.T) {
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	var left, right bytes.Buffer

	if err := Generate(&left, startDate); err != nil {
		t.Fatalf("generate left: %v", err)
	}
	if err := Generate(&right, startDate); err != nil {
		t.Fatalf("generate right: %v", err)
	}
	if left.String() != right.String() {
		t.Fatal("fixed-date seed output is not reproducible")
	}
}

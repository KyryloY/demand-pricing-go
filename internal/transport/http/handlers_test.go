package httptransport

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDemandUnknownStoreReturnsConsistentNotFound(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/stores/UNKNOWN/products/DRILL-18V/demand", nil)

	NewRouter(Dependencies{}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
	if got, want := recorder.Body.String(), `{"error":{"code":"not_found","message":"store not found"}}`+"\n"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

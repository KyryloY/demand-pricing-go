package httptransport

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/KyryloY/demand-pricing-go/internal/domain/catalog"
	"github.com/KyryloY/demand-pricing-go/internal/observability"
	"github.com/KyryloY/demand-pricing-go/internal/service"
)

type CatalogReader interface {
	ListStores(ctx context.Context) ([]catalog.Store, error)
	ListProducts(ctx context.Context, search, category string, limit, offset int) ([]catalog.Product, error)
	Product(ctx context.Context, sku string) (catalog.Product, error)
}

type Dependencies struct {
	Catalog        CatalogReader
	Importer       *service.SalesImporter
	Forecast       *service.ForecastService
	Recommendation *service.RecommendationService
	Ready          func(context.Context) error
}

func NewRouter(dependencies Dependencies) http.Handler {
	router := chi.NewRouter()
	router.Use(observability.RequestID)
	router.Use(observability.Recoverer)
	router.Use(observability.Metrics)
	router.Get("/", dashboard)
	router.Get("/healthz", health)
	router.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if dependencies.Ready != nil {
			if err := dependencies.Ready(r.Context()); err != nil {
				writeError(w, http.StatusServiceUnavailable, "not_ready", "database is not ready")
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	router.Handle("/metrics", promhttp.Handler())
	router.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	router.Route("/api/v1", func(router chi.Router) {
		router.Get("/stores", func(w http.ResponseWriter, r *http.Request) {
			if dependencies.Catalog == nil {
				writeJSON(w, http.StatusOK, []catalog.Store{})
				return
			}
			stores, err := dependencies.Catalog.ListStores(r.Context())
			if err != nil {
				writeError(w, http.StatusInternalServerError, "internal_error", "could not load stores")
				return
			}
			writeJSON(w, http.StatusOK, stores)
		})
		router.Get("/products", func(w http.ResponseWriter, r *http.Request) {
			if dependencies.Catalog == nil {
				writeJSON(w, http.StatusOK, []catalog.Product{})
				return
			}
			limit, offset := queryPagination(r)
			products, err := dependencies.Catalog.ListProducts(r.Context(), r.URL.Query().Get("search"), r.URL.Query().Get("category"), limit, offset)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "internal_error", "could not load products")
				return
			}
			writeJSON(w, http.StatusOK, products)
		})
		router.Get("/products/{sku}", func(w http.ResponseWriter, r *http.Request) {
			if dependencies.Catalog == nil {
				writeError(w, http.StatusNotFound, "not_found", "product not found")
				return
			}
			product, err := dependencies.Catalog.Product(r.Context(), chi.URLParam(r, "sku"))
			if err != nil {
				writeError(w, http.StatusNotFound, "not_found", "product not found")
				return
			}
			writeJSON(w, http.StatusOK, product)
		})
		router.Get("/stores/{storeCode}/products/{sku}/demand", func(w http.ResponseWriter, r *http.Request) {
			if dependencies.Forecast == nil {
				writeError(w, http.StatusNotFound, "not_found", "store not found")
				return
			}
			forecast, err := dependencies.Forecast.Recalculate(r.Context(), chi.URLParam(r, "storeCode"), chi.URLParam(r, "sku"), requestDate(r))
			if err != nil {
				writeError(w, http.StatusNotFound, "not_found", notFoundMessage(err, "store not found"))
				return
			}
			writeJSON(w, http.StatusOK, forecast)
		})
		router.Get("/stores/{storeCode}/products/{sku}/recommendation", func(w http.ResponseWriter, r *http.Request) {
			if dependencies.Recommendation == nil {
				writeError(w, http.StatusNotFound, "not_found", "store not found")
				return
			}
			recommendation, err := dependencies.Recommendation.Recalculate(r.Context(), chi.URLParam(r, "storeCode"), chi.URLParam(r, "sku"), requestDate(r))
			if err != nil {
				writeError(w, http.StatusNotFound, "not_found", notFoundMessage(err, "store not found"))
				return
			}
			writeJSON(w, http.StatusOK, recommendation)
		})
		router.Post("/forecasts/recalculate", recalculateForecast(dependencies.Forecast))
		router.Post("/recommendations/recalculate", recalculateRecommendation(dependencies.Recommendation))
		router.Post("/imports/sales", importSales(dependencies.Importer))
	})
	router.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "route not found")
	})
	return router
}

func recalculateForecast(forecastService *service.ForecastService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if forecastService == nil {
			writeError(w, http.StatusServiceUnavailable, "unavailable", "forecast service is unavailable")
			return
		}
		var request recalculationRequest
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		forecast, err := forecastService.Recalculate(r.Context(), request.StoreCode, request.SKU, request.Date())
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "calculation_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, forecast)
	}
}

func recalculateRecommendation(recommendationService *service.RecommendationService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if recommendationService == nil {
			writeError(w, http.StatusServiceUnavailable, "unavailable", "recommendation service is unavailable")
			return
		}
		var request recalculationRequest
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		recommendation, err := recommendationService.Recalculate(r.Context(), request.StoreCode, request.SKU, request.Date())
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "calculation_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, recommendation)
	}
}

func importSales(importer *service.SalesImporter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if importer == nil {
			writeError(w, http.StatusServiceUnavailable, "unavailable", "sales importer is unavailable")
			return
		}
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "multipart CSV is required")
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "multipart field file is required")
			return
		}
		defer file.Close()
		summary, err := importer.Import(r.Context(), file)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "import_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, summary)
	}
}

type recalculationRequest struct {
	StoreCode string `json:"store_code"`
	SKU       string `json:"sku"`
	DateValue string `json:"date"`
}

func (r recalculationRequest) Date() time.Time {
	if r.DateValue == "" {
		return time.Now().UTC().Truncate(24 * time.Hour)
	}
	date, err := time.Parse("2006-01-02", r.DateValue)
	if err != nil {
		return time.Time{}
	}
	return date
}

func requestDate(r *http.Request) time.Time {
	if value := r.URL.Query().Get("date"); value != "" {
		if date, err := time.Parse("2006-01-02", value); err == nil {
			return date
		}
	}
	return time.Now().UTC().Truncate(24 * time.Hour)
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	if err := decoder.Decode(target); err != nil {
		return errors.New("invalid JSON body")
	}
	return nil
}

func queryPagination(r *http.Request) (int, int) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func notFoundMessage(err error, fallback string) string {
	if strings.Contains(strings.ToLower(err.Error()), "product") {
		return "product not found"
	}
	return fallback
}

func health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func dashboard(w http.ResponseWriter, _ *http.Request) {
	const page = `<!doctype html><html><head><meta charset="utf-8"><title>Demand Pricing</title></head><body><main><h1>Demand Pricing</h1><p>Deterministic synthetic demand and price recommendations.</p></main></body></html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = template.New("dashboard").Parse(page)
	_, _ = w.Write([]byte(page))
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{Error: errorBody{Code: code, Message: message}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

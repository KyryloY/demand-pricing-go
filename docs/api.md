# API

All API responses are JSON unless the root dashboard is requested. The data is synthetic and deterministic.

## Operational endpoints

- `GET /healthz` — process health.
- `GET /readyz` — PostgreSQL readiness.
- `GET /metrics` — Prometheus metrics.
- `GET /` — dashboard shell.

## Catalog

- `GET /api/v1/stores`
- `GET /api/v1/products?search=&category=&limit=&offset=`
- `GET /api/v1/products/{sku}`

## Demand and recommendations

- `GET /api/v1/stores/{storeCode}/products/{sku}/demand?date=YYYY-MM-DD`
- `GET /api/v1/stores/{storeCode}/products/{sku}/recommendation?date=YYYY-MM-DD`
- `POST /api/v1/forecasts/recalculate`
- `POST /api/v1/recommendations/recalculate`

Recalculation bodies are `{ "store_code": "MANNHEIM-01", "sku": "DRILL-18V", "date": "2025-04-30" }`; `date` is optional.

## Sales import

`POST /api/v1/imports/sales` accepts a multipart field named `file` containing CSV columns `store_code,sku,sale_date,units_sold,unit_price,promotion_code`.

Errors use the stable shape:

```json
{"error":{"code":"not_found","message":"store not found"}}
```

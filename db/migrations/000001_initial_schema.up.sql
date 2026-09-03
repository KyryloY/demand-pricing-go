CREATE TABLE stores (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    country_code CHAR(2) NOT NULL DEFAULT 'DE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT stores_code_not_blank CHECK (btrim(code) <> ''),
    CONSTRAINT stores_country_code_format CHECK (country_code ~ '^[A-Z]{2}$')
);

CREATE TABLE products (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sku TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    cost_price NUMERIC(12, 2) NOT NULL CHECK (cost_price >= 0),
    min_margin_pct NUMERIC(5, 4) NOT NULL CHECK (min_margin_pct >= 0 AND min_margin_pct <= 1),
    min_price NUMERIC(12, 2) NOT NULL CHECK (min_price >= 0),
    max_price NUMERIC(12, 2) NOT NULL CHECK (max_price >= min_price),
    price_elasticity NUMERIC(8, 4) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT products_sku_not_blank CHECK (btrim(sku) <> '')
);

CREATE TABLE inventory_snapshots (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    store_id BIGINT NOT NULL REFERENCES stores (id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    snapshot_date DATE NOT NULL,
    on_hand_units INTEGER NOT NULL CHECK (on_hand_units >= 0),
    UNIQUE (store_id, product_id, snapshot_date)
);

CREATE TABLE promotions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    product_id BIGINT NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    store_id BIGINT REFERENCES stores (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    discount_pct NUMERIC(5, 4) NOT NULL CHECK (discount_pct >= 0 AND discount_pct <= 1),
    starts_on DATE NOT NULL,
    ends_on DATE NOT NULL,
    CONSTRAINT promotions_dates_valid CHECK (ends_on >= starts_on),
    CONSTRAINT promotions_code_not_blank CHECK (btrim(code) <> '')
);

CREATE TABLE daily_sales (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    store_id BIGINT NOT NULL REFERENCES stores (id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    sale_date DATE NOT NULL,
    units_sold INTEGER NOT NULL CHECK (units_sold >= 0),
    revenue NUMERIC(12, 2) NOT NULL CHECK (revenue >= 0),
    unit_price NUMERIC(12, 2) NOT NULL CHECK (unit_price >= 0),
    promotion_id BIGINT REFERENCES promotions (id) ON DELETE SET NULL,
    UNIQUE (store_id, product_id, sale_date)
);

CREATE TABLE demand_forecasts (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    store_id BIGINT NOT NULL REFERENCES stores (id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    forecast_date DATE NOT NULL,
    forecast_daily_units NUMERIC(12, 4) NOT NULL CHECK (forecast_daily_units >= 0),
    trend_pct NUMERIC(8, 4) NOT NULL,
    confidence NUMERIC(5, 4) NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (store_id, product_id, forecast_date)
);

CREATE TABLE price_recommendations (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    store_id BIGINT NOT NULL REFERENCES stores (id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    recommendation_date DATE NOT NULL,
    current_price NUMERIC(12, 2) NOT NULL CHECK (current_price >= 0),
    recommended_price NUMERIC(12, 2) NOT NULL CHECK (recommended_price >= 0),
    expected_daily_units NUMERIC(12, 4) NOT NULL CHECK (expected_daily_units >= 0),
    expected_daily_margin NUMERIC(12, 2) NOT NULL CHECK (expected_daily_margin >= 0),
    status TEXT NOT NULL,
    reason_codes JSONB NOT NULL DEFAULT '[]'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT price_recommendations_reason_codes_array CHECK (jsonb_typeof(reason_codes) = 'array'),
    UNIQUE (store_id, product_id, recommendation_date)
);

CREATE INDEX daily_sales_lookup_idx
    ON daily_sales (store_id, product_id, sale_date DESC);

CREATE INDEX inventory_snapshots_lookup_idx
    ON inventory_snapshots (store_id, product_id, snapshot_date DESC);

CREATE INDEX demand_forecasts_lookup_idx
    ON demand_forecasts (store_id, product_id, forecast_date DESC);

CREATE INDEX price_recommendations_lookup_idx
    ON price_recommendations (store_id, product_id, recommendation_date DESC);

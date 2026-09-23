ALTER TABLE orders
ADD COLUMN discount_code TEXT,
ADD COLUMN discount_amount BIGINT NOT NULL DEFAULT 0;

CREATE TABLE discount_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    code TEXT NOT NULL UNIQUE,

    discount_type TEXT NOT NULL
        CHECK (discount_type IN ('percentage', 'fixed')),

    discount_value BIGINT NOT NULL
        CHECK (discount_value > 0),

    min_order_amount BIGINT NOT NULL DEFAULT 0
        CHECK (min_order_amount >= 0),

    max_discount_amount BIGINT,

    usage_limit INT,
    used_count INT NOT NULL DEFAULT 0,

    starts_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,

    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
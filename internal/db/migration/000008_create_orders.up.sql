CREATE TABLE shipping_settings (
    id      INT PRIMARY KEY DEFAULT 1,
    cost    BIGINT NOT NULL DEFAULT 0,
    CHECK (id = 1)
);

INSERT INTO shipping_settings (id, cost) VALUES (1, 0);

CREATE TABLE orders (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES users(id),
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    shipping_cost   BIGINT      NOT NULL,
    total_price     BIGINT      NOT NULL,
    -- address snapshot
    address_title       VARCHAR(50),
    address_province    VARCHAR(50),
    address_city        VARCHAR(50),
    address_detail      TEXT,
    address_postal_code VARCHAR(10),
    address_lat         NUMERIC(9,6),
    address_lng         NUMERIC(9,6),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE order_items (
    id          UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id    UUID    NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    variant_id  UUID    NOT NULL REFERENCES product_variants(id),
    product_id  UUID    NOT NULL REFERENCES products(id),
    quantity    INT     NOT NULL CHECK (quantity > 0),
    unit_price  BIGINT  NOT NULL,
    total_price BIGINT  NOT NULL
);
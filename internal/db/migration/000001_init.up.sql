CREATE TABLE users (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    full_name   VARCHAR(100) NOT NULL,
    phone       VARCHAR(20)  NOT NULL UNIQUE,
    password    TEXT         NOT NULL,
    created_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sessions (
    id            UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    phone         VARCHAR NOT NULL,
    refresh_token TEXT    NOT NULL UNIQUE,
    user_agent    TEXT,
    client_ip     INET,
    is_blocked    BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at    TIMESTAMP NOT NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE categories (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(100) NOT NULL,
    parent_id  UUID         REFERENCES categories(id) ON DELETE SET NULL,
    image_url  TEXT,
    created_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE products (
    id          UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(200)  NOT NULL,
    description TEXT,
    price       NUMERIC(12,2) NOT NULL,
    discount    NUMERIC(5,2)  DEFAULT 0,
    category_id UUID          REFERENCES categories(id) ON DELETE SET NULL,
    is_active   BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE product_variants (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID        NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    label      VARCHAR(50) NOT NULL,
    color      VARCHAR(50),
    stock      INTEGER     NOT NULL DEFAULT 0,
    created_at TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE product_images (
    id         UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID    NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    url        TEXT    NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE reviews (
    id         BIGSERIAL   PRIMARY KEY,
    product_id UUID        NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    rating     SMALLINT    CHECK (rating BETWEEN 1 AND 5),
    body       TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (product_id, user_id),
    CHECK (rating IS NOT NULL OR body IS NOT NULL)
);

CREATE TABLE user_addresses (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       VARCHAR(50),
    province    VARCHAR(50),
    city        VARCHAR(50),
    address     TEXT,
    postal_code VARCHAR(10),
    lat         NUMERIC(9,6),
    lng         NUMERIC(9,6),
    is_default  BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE cart_items (
    id         UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID    NOT NULL REFERENCES users(id)             ON DELETE CASCADE,
    variant_id UUID    NOT NULL REFERENCES product_variants(id)  ON DELETE CASCADE,
    quantity   INTEGER NOT NULL DEFAULT 1 CHECK (quantity > 0),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, variant_id)
);

CREATE TABLE shipping_settings (
    id   INT    PRIMARY KEY DEFAULT 1,
    cost BIGINT NOT NULL DEFAULT 0,
    CHECK (id = 1)
);

CREATE TABLE orders (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID        NOT NULL REFERENCES users(id),
    status              VARCHAR(20) NOT NULL DEFAULT 'pending',
    shipping_cost       BIGINT      NOT NULL,
    total_price         BIGINT      NOT NULL,
    address_title       VARCHAR(50),
    address_province    VARCHAR(50),
    address_city        VARCHAR(50),
    address_detail      TEXT,
    address_postal_code VARCHAR(10),
    address_lat         NUMERIC(9,6),
    address_lng         NUMERIC(9,6),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE order_items (
    id          UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id    UUID    NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    variant_id  UUID    REFERENCES product_variants(id) ON DELETE SET NULL,
    product_id  UUID    REFERENCES products(id)         ON DELETE SET NULL,
    quantity    INT     NOT NULL CHECK (quantity > 0),
    unit_price  BIGINT  NOT NULL,
    total_price BIGINT  NOT NULL
);

-- seed data
INSERT INTO shipping_settings (id, cost) VALUES (1, 0);

INSERT INTO users (full_name, phone, password) VALUES
    ('فرهاد امیرانی', '09927253853', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi');

INSERT INTO categories (name) VALUES
    ('فوتبال'),
    ('کشتی'),
    ('والیبال'),
    ('ورزش های راکتی'),
    ('تمرین و هوازی'),
    ('اکسسوری');
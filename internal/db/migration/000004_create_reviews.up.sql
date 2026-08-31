
CREATE TABLE reviews (
    id         BIGSERIAL    PRIMARY KEY,
    product_id UUID         NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    user_id    UUID         NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    rating     SMALLINT     CHECK (rating BETWEEN 1 AND 5),
    body       TEXT,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (product_id, user_id),
    CHECK (rating IS NOT NULL OR body IS NOT NULL)
);
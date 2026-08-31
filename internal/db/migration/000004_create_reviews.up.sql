
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

INSERT INTO reviews (product_id, user_id, rating, body)
VALUES (
    'd1ae257e-36f1-4066-bb9d-6a84f6ecc62d',
    (SELECT id FROM users LIMIT 1),
    4,
    'محصول خوبیه، کیفیت عالی داره'
);
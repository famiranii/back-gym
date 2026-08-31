
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
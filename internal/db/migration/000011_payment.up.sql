CREATE TABLE payments (
    id BIGSERIAL PRIMARY KEY,

    order_id UUID NOT NULL REFERENCES orders(id),

    amount BIGINT NOT NULL,
    authority TEXT NOT NULL,

    status TEXT NOT NULL DEFAULT 'pending',

    ref_id BIGINT,
    card_pan TEXT,
    card_hash TEXT,
    fee_type TEXT,
    fee BIGINT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
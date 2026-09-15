CREATE TABLE otp_codes (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    phone        VARCHAR(20)  NOT NULL,
    code         VARCHAR(10)  NOT NULL,
    purpose      VARCHAR(20)  NOT NULL,
    full_name    VARCHAR(100),
    password     TEXT,
    expires_at   TIMESTAMP    NOT NULL,
    consumed_at  TIMESTAMP,
    attempts     INT          NOT NULL DEFAULT 0,
    created_at   TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_otp_codes_phone_purpose ON otp_codes (phone, purpose);

-- down

ALTER TABLE wishlists
DROP CONSTRAINT wishlists_pkey;

ALTER TABLE wishlists
ADD COLUMN id UUID NOT NULL DEFAULT gen_random_uuid();

ALTER TABLE wishlists
ADD CONSTRAINT wishlists_pkey PRIMARY KEY (id);
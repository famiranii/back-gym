ALTER TABLE wishlists
DROP CONSTRAINT wishlists_pkey;

ALTER TABLE wishlists
DROP COLUMN id;

ALTER TABLE wishlists
ADD CONSTRAINT wishlists_pkey PRIMARY KEY (user_id, product_id);
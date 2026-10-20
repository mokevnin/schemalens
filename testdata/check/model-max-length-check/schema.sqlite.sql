CREATE TABLE products (
  id BIGINT NOT NULL PRIMARY KEY,
  sku TEXT CHECK (length(sku) <= 12),
  label VARCHAR(255) CHECK (length(label) <= 50)
);

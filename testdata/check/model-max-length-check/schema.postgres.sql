CREATE TABLE products (
  id BIGINT NOT NULL PRIMARY KEY,
  sku TEXT CHECK (char_length(sku) <= 12),
  label VARCHAR(255) CHECK (char_length(label) <= 50)
);

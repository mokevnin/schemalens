CREATE TABLE products (
  id INTEGER PRIMARY KEY,
  sku TEXT CHECK (length(sku) <= 12),
  label VARCHAR(255) CHECK (length(label) <= 50)
);

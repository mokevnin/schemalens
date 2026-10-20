CREATE TABLE products (
  id INTEGER PRIMARY KEY,
  sku TEXT CHECK (char_length(sku) <= 12),
  label VARCHAR(255) CHECK (char_length(label) <= 50)
);

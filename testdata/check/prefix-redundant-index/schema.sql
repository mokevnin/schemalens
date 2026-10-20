CREATE TABLE orders (
  id BIGINT NOT NULL PRIMARY KEY,
  customer_id BIGINT NOT NULL,
  status VARCHAR(20) NOT NULL,
  placed_at VARCHAR(30) NOT NULL
);
CREATE INDEX orders_customer_idx ON orders (customer_id);
CREATE INDEX orders_customer_status_idx ON orders (customer_id, status);
CREATE UNIQUE INDEX orders_placed_uq ON orders (placed_at, status);
CREATE INDEX orders_placed_idx ON orders (placed_at);
CREATE TABLE memberships (
  user_id BIGINT NOT NULL,
  group_id BIGINT NOT NULL,
  PRIMARY KEY (user_id, group_id)
);
CREATE INDEX memberships_user_idx ON memberships (user_id);

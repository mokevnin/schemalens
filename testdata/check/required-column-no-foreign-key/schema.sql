CREATE TABLE tenants (id INTEGER PRIMARY KEY);
CREATE TABLE orders (
  id INTEGER PRIMARY KEY,
  tenant_id INTEGER NOT NULL
);
CREATE INDEX orders_tenant_idx ON orders (tenant_id);

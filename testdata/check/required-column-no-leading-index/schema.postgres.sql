CREATE TABLE tenants (id INTEGER PRIMARY KEY);
CREATE TABLE orders (
  id INTEGER PRIMARY KEY,
  tenant_id INTEGER NOT NULL REFERENCES tenants (id),
  total INTEGER
);
CREATE INDEX orders_total_tenant_idx ON orders (total, tenant_id);

CREATE TABLE tenants (id BIGINT NOT NULL PRIMARY KEY);
CREATE TABLE orders (
  id BIGINT NOT NULL PRIMARY KEY,
  tenant_id BIGINT NOT NULL REFERENCES tenants (id),
  total INTEGER
);
CREATE INDEX orders_total_tenant_idx ON orders (total, tenant_id);

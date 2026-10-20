CREATE TABLE tenants (id BIGINT NOT NULL PRIMARY KEY);
CREATE TABLE orders (
  id BIGINT NOT NULL PRIMARY KEY,
  tenant_id BIGINT,
  FOREIGN KEY (tenant_id) REFERENCES tenants (id)
);
CREATE INDEX orders_tenant_idx ON orders (tenant_id);

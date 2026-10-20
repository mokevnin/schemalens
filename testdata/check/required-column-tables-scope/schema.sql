CREATE TABLE tenants (id BIGINT NOT NULL PRIMARY KEY);
CREATE TABLE orders (
  id BIGINT NOT NULL PRIMARY KEY,
  tenant_id BIGINT NOT NULL,
  FOREIGN KEY (tenant_id) REFERENCES tenants (id)
);
CREATE INDEX orders_tenant_idx ON orders (tenant_id);
CREATE TABLE audit_log (id BIGINT NOT NULL PRIMARY KEY, message TEXT);
CREATE TABLE invoices (id BIGINT NOT NULL PRIMARY KEY);

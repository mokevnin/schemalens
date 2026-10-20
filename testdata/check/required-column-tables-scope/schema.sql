CREATE TABLE tenants (id INTEGER PRIMARY KEY);
CREATE TABLE orders (
  id INTEGER PRIMARY KEY,
  tenant_id INTEGER NOT NULL,
  FOREIGN KEY (tenant_id) REFERENCES tenants (id)
);
CREATE INDEX orders_tenant_idx ON orders (tenant_id);
CREATE TABLE audit_log (id INTEGER PRIMARY KEY, message TEXT);
CREATE TABLE invoices (id INTEGER PRIMARY KEY);

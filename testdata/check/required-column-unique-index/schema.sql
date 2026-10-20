CREATE TABLE tenants (id INTEGER PRIMARY KEY);
CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  tenant_id INTEGER NOT NULL,
  email VARCHAR(255) NOT NULL,
  handle VARCHAR(255) NOT NULL,
  FOREIGN KEY (tenant_id) REFERENCES tenants (id)
);
CREATE INDEX users_tenant_idx ON users (tenant_id);
CREATE UNIQUE INDEX users_email_uniq ON users (email);
CREATE UNIQUE INDEX users_tenant_handle_uniq ON users (tenant_id, handle);

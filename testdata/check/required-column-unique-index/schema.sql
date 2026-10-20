CREATE TABLE tenants (id INTEGER PRIMARY KEY);
CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  tenant_id INTEGER NOT NULL REFERENCES tenants (id),
  email TEXT NOT NULL,
  handle TEXT NOT NULL
);
CREATE INDEX users_tenant_idx ON users (tenant_id);
CREATE UNIQUE INDEX users_email_uniq ON users (email);
CREATE UNIQUE INDEX users_tenant_handle_uniq ON users (tenant_id, handle);

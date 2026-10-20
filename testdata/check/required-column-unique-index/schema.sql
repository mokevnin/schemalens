CREATE TABLE tenants (id BIGINT NOT NULL PRIMARY KEY);
CREATE TABLE users (
  id BIGINT NOT NULL PRIMARY KEY,
  tenant_id BIGINT NOT NULL,
  email VARCHAR(255) NOT NULL,
  handle VARCHAR(255) NOT NULL,
  FOREIGN KEY (tenant_id) REFERENCES tenants (id)
);
CREATE UNIQUE INDEX users_email_uniq ON users (email);
CREATE UNIQUE INDEX users_tenant_handle_uniq ON users (tenant_id, handle);

CREATE TABLE users (
  id BIGINT NOT NULL PRIMARY KEY,
  email VARCHAR(100) NOT NULL,
  name VARCHAR(100) NOT NULL
);
CREATE INDEX users_email_idx ON users (email);
CREATE INDEX users_email_copy_idx ON users (email);
CREATE UNIQUE INDEX users_name_uq ON users (name, email);
CREATE UNIQUE INDEX users_name_uq_copy ON users (name, email);

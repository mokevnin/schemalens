CREATE TABLE accounts (
  id BIGINT NOT NULL PRIMARY KEY,
  email VARCHAR(100) NOT NULL,
  region VARCHAR(20) NOT NULL
);
CREATE UNIQUE INDEX accounts_email_uq ON accounts (email);
CREATE INDEX accounts_email_region_idx ON accounts (email, region);
CREATE INDEX accounts_region_idx ON accounts (region);

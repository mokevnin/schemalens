CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  email VARCHAR(100) NOT NULL,
  handle VARCHAR(50) UNIQUE,
  slug VARCHAR(50)
);
CREATE INDEX users_email ON users (email);
CREATE TABLE memberships (
  id INTEGER PRIMARY KEY,
  org_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL
);
CREATE INDEX memberships_pair ON memberships (org_id, user_id);

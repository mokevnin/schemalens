CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  email TEXT NOT NULL
);
CREATE UNIQUE INDEX users_email_active ON users (email) WHERE id > 0;

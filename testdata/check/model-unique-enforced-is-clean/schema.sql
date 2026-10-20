CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  email VARCHAR(100) NOT NULL UNIQUE,
  handle VARCHAR(50)
);
CREATE UNIQUE INDEX users_handle ON users (handle);
CREATE TABLE memberships (
  id INTEGER PRIMARY KEY,
  org_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL
);
CREATE UNIQUE INDEX memberships_pair ON memberships (org_id, user_id);
CREATE TABLE user_roles (
  user_id INTEGER NOT NULL,
  role_id INTEGER NOT NULL,
  PRIMARY KEY (user_id, role_id)
);

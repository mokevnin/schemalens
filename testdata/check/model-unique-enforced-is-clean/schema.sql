CREATE TABLE users (
  id BIGINT NOT NULL PRIMARY KEY,
  email VARCHAR(100) NOT NULL UNIQUE,
  handle VARCHAR(50)
);
CREATE UNIQUE INDEX users_handle ON users (handle);
CREATE TABLE memberships (
  id BIGINT NOT NULL PRIMARY KEY,
  org_id BIGINT NOT NULL,
  user_id BIGINT NOT NULL
);
CREATE UNIQUE INDEX memberships_pair ON memberships (org_id, user_id);
CREATE TABLE user_roles (
  user_id BIGINT NOT NULL,
  role_id BIGINT NOT NULL,
  PRIMARY KEY (user_id, role_id)
);

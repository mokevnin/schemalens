CREATE TABLE users (id INTEGER PRIMARY KEY);
CREATE TABLE roles (id INTEGER PRIMARY KEY);
CREATE TABLE user_roles (
  user_id INTEGER NOT NULL REFERENCES users (id),
  role_id INTEGER NOT NULL REFERENCES roles (id),
  PRIMARY KEY (user_id, role_id)
);
CREATE INDEX user_roles_role_id ON user_roles (role_id);

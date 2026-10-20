CREATE TABLE users (id INTEGER PRIMARY KEY);
CREATE TABLE roles (id INTEGER PRIMARY KEY);
CREATE TABLE user_roles (
  user_id INTEGER NOT NULL,
  role_id INTEGER NOT NULL,
  PRIMARY KEY (user_id, role_id),
  FOREIGN KEY (user_id) REFERENCES users (id),
  FOREIGN KEY (role_id) REFERENCES roles (id)
);
CREATE INDEX user_roles_role_id ON user_roles (role_id);

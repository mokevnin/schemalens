CREATE TABLE users (id INTEGER PRIMARY KEY);
CREATE TABLE roles (id INTEGER PRIMARY KEY);
CREATE TABLE posts (
  id INTEGER PRIMARY KEY,
  author_id INTEGER NOT NULL,
  editor_id INTEGER,
  FOREIGN KEY (editor_id) REFERENCES users (id)
);
CREATE TABLE user_roles (
  user_id INTEGER NOT NULL,
  role_id INTEGER NOT NULL,
  PRIMARY KEY (user_id, role_id),
  FOREIGN KEY (user_id) REFERENCES users (id)
);
CREATE INDEX posts_author ON posts (author_id);
CREATE INDEX posts_editor ON posts (editor_id);
CREATE INDEX user_roles_role ON user_roles (role_id);

CREATE TABLE users (id BIGINT NOT NULL PRIMARY KEY);
CREATE TABLE roles (id BIGINT NOT NULL PRIMARY KEY);
CREATE TABLE posts (
  id BIGINT NOT NULL PRIMARY KEY,
  author_id BIGINT NOT NULL,
  editor_id BIGINT,
  FOREIGN KEY (editor_id) REFERENCES users (id)
);
CREATE TABLE user_roles (
  user_id BIGINT NOT NULL,
  role_id BIGINT NOT NULL,
  PRIMARY KEY (user_id, role_id),
  FOREIGN KEY (user_id) REFERENCES users (id)
);
CREATE INDEX posts_author ON posts (author_id);
CREATE INDEX posts_editor ON posts (editor_id);
CREATE INDEX user_roles_role ON user_roles (role_id);

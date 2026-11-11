CREATE TABLE accounts (id BIGINT NOT NULL PRIMARY KEY);
CREATE TABLE users (
  id BIGINT NOT NULL PRIMARY KEY,
  account_id BIGINT NOT NULL,
  FOREIGN KEY (account_id) REFERENCES accounts (id)
);
CREATE UNIQUE INDEX users_account_id_id ON users (account_id, id);
CREATE TABLE posts (
  id BIGINT NOT NULL PRIMARY KEY,
  account_id BIGINT NOT NULL,
  scope_id BIGINT,
  author_id BIGINT NOT NULL,
  editor_id BIGINT,
  FOREIGN KEY (account_id) REFERENCES accounts (id),
  FOREIGN KEY (account_id, author_id) REFERENCES users (account_id, id),
  FOREIGN KEY (scope_id, editor_id) REFERENCES users (account_id, id)
);
CREATE INDEX posts_account_author ON posts (account_id, author_id);
CREATE INDEX posts_scope_editor ON posts (scope_id, editor_id);

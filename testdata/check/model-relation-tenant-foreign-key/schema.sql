CREATE TABLE accounts (id BIGINT NOT NULL PRIMARY KEY);
CREATE TABLE users (
  id BIGINT NOT NULL PRIMARY KEY,
  account_id BIGINT NOT NULL,
  FOREIGN KEY (account_id) REFERENCES accounts (id)
);
CREATE UNIQUE INDEX users_account_id_id ON users (account_id, id);
CREATE TABLE teams (
  account_id BIGINT NOT NULL,
  id BIGINT NOT NULL,
  PRIMARY KEY (account_id, id),
  FOREIGN KEY (account_id) REFERENCES accounts (id)
);
CREATE UNIQUE INDEX teams_id_account_id ON teams (id, account_id);
CREATE TABLE posts (
  id BIGINT NOT NULL PRIMARY KEY,
  account_id BIGINT NOT NULL,
  scope_id BIGINT,
  author_id BIGINT NOT NULL,
  editor_id BIGINT,
  team_id BIGINT NOT NULL,
  owner_team_id BIGINT NOT NULL,
  reviewer_team_id BIGINT NOT NULL,
  FOREIGN KEY (account_id) REFERENCES accounts (id),
  FOREIGN KEY (account_id, author_id) REFERENCES users (account_id, id),
  FOREIGN KEY (scope_id, editor_id) REFERENCES users (account_id, id),
  FOREIGN KEY (account_id, team_id) REFERENCES teams (account_id, id),
  FOREIGN KEY (account_id, owner_team_id) REFERENCES teams (id, account_id),
  FOREIGN KEY (account_id, reviewer_team_id) REFERENCES teams (id, account_id)
);
CREATE INDEX posts_account_author ON posts (account_id, author_id);
CREATE INDEX posts_scope_editor ON posts (scope_id, editor_id);
CREATE INDEX posts_account_team ON posts (account_id, team_id);
CREATE INDEX posts_account_owner_team ON posts (account_id, owner_team_id);
CREATE INDEX posts_account_reviewer_team ON posts (account_id, reviewer_team_id);

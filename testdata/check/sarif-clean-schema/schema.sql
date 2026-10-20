CREATE TABLE users (
  id BIGINT NOT NULL PRIMARY KEY,
  email TEXT NOT NULL
);
CREATE TABLE memberships (
  user_id BIGINT NOT NULL,
  team_id BIGINT NOT NULL,
  PRIMARY KEY (user_id, team_id)
);

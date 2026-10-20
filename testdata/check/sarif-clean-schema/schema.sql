CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  email TEXT NOT NULL
);
CREATE TABLE memberships (
  user_id INTEGER NOT NULL,
  team_id INTEGER NOT NULL,
  PRIMARY KEY (user_id, team_id)
);

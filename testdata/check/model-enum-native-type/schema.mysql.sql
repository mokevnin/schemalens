CREATE TABLE posts (
  id INTEGER PRIMARY KEY,
  status ENUM('draft', 'published') NOT NULL,
  kind ENUM('a', 'b', 'c')
);

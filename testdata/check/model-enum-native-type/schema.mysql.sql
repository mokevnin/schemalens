CREATE TABLE posts (
  id BIGINT NOT NULL PRIMARY KEY,
  status ENUM('draft', 'published') NOT NULL,
  kind ENUM('a', 'b', 'c')
);

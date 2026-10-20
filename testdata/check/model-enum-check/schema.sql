CREATE TABLE posts (
  id BIGINT NOT NULL PRIMARY KEY,
  status VARCHAR(20) NOT NULL CHECK (status IN ('draft', 'published')),
  kind VARCHAR(20) CHECK (kind IN ('a', 'b', 'c'))
);

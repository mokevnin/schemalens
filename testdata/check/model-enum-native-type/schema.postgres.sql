CREATE TYPE post_status AS ENUM ('draft', 'published');
CREATE TYPE post_kind AS ENUM ('a', 'b', 'c');
CREATE TABLE posts (
  id INTEGER PRIMARY KEY,
  status post_status NOT NULL,
  kind post_kind
);

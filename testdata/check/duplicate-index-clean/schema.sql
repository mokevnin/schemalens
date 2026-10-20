CREATE TABLE users (
  id BIGINT NOT NULL PRIMARY KEY,
  email VARCHAR(100) NOT NULL,
  name VARCHAR(100) NOT NULL
);
CREATE INDEX users_name_email_idx ON users (name, email);
CREATE INDEX users_email_name_idx ON users (email, name);
CREATE INDEX users_name_desc_email_idx ON users (name DESC, email);
CREATE TABLE posts (
  id BIGINT NOT NULL PRIMARY KEY,
  author_id BIGINT NOT NULL
);
CREATE INDEX posts_author_idx ON posts (author_id);
CREATE INDEX users_id_name_idx ON users (id, name);

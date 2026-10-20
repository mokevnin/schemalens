CREATE TABLE parent (
  id SERIAL PRIMARY KEY
);
CREATE TABLE small (
  id SMALLINT NOT NULL PRIMARY KEY
);
CREATE TABLE child (
  id BIGSERIAL PRIMARY KEY,
  parent_id INTEGER NOT NULL,
  CONSTRAINT child_parent_fk FOREIGN KEY (parent_id) REFERENCES parent (id)
);
CREATE INDEX child_parent_idx ON child (parent_id);

CREATE TABLE parent (
  id INTEGER NOT NULL PRIMARY KEY
);
CREATE TABLE child (
  id INTEGER NOT NULL PRIMARY KEY,
  parent_id INT NOT NULL,
  CONSTRAINT child_parent_fk FOREIGN KEY (parent_id) REFERENCES parent (id)
);
CREATE INDEX child_parent_idx ON child (parent_id);

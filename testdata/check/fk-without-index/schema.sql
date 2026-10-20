CREATE TABLE parent (
  id INTEGER PRIMARY KEY
);
CREATE TABLE child (
  id INTEGER PRIMARY KEY,
  parent_id INTEGER NOT NULL,
  CONSTRAINT child_parent_fk FOREIGN KEY (parent_id) REFERENCES parent (id)
);

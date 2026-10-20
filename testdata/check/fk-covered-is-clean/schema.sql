CREATE TABLE parent (
  id INTEGER PRIMARY KEY,
  code VARCHAR(20) NOT NULL UNIQUE
);
CREATE TABLE by_index (
  id INTEGER PRIMARY KEY,
  parent_id INTEGER NOT NULL,
  CONSTRAINT by_index_parent_fk FOREIGN KEY (parent_id) REFERENCES parent (id)
);
CREATE INDEX by_index_parent_idx ON by_index (parent_id);
CREATE TABLE by_primary_key (
  parent_id INTEGER NOT NULL,
  seq INTEGER NOT NULL,
  PRIMARY KEY (parent_id, seq),
  CONSTRAINT by_pk_parent_fk FOREIGN KEY (parent_id) REFERENCES parent (id)
);
CREATE TABLE by_unique (
  id INTEGER PRIMARY KEY,
  parent_code VARCHAR(20) NOT NULL UNIQUE,
  CONSTRAINT by_unique_parent_fk FOREIGN KEY (parent_code) REFERENCES parent (code)
);
CREATE TABLE by_composite_index (
  id INTEGER PRIMARY KEY,
  parent_id INTEGER NOT NULL,
  seq INTEGER NOT NULL,
  CONSTRAINT by_composite_fk FOREIGN KEY (parent_id) REFERENCES parent (id)
);
CREATE INDEX by_composite_idx ON by_composite_index (parent_id, seq);

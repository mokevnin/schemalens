CREATE TABLE parent (
  id BIGINT NOT NULL PRIMARY KEY
);
CREATE TABLE child (
  id BIGINT NOT NULL PRIMARY KEY,
  parent_id BIGINT NOT NULL,
  quantity INT NOT NULL,
  CONSTRAINT child_parent_fk FOREIGN KEY (parent_id) REFERENCES parent (id)
);
CREATE INDEX child_parent_idx ON child (parent_id);
CREATE TABLE tag_link (
  tag_id INT NOT NULL,
  item_id INT NOT NULL,
  PRIMARY KEY (tag_id, item_id)
);

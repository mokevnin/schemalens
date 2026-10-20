CREATE TABLE users (id INTEGER PRIMARY KEY);
CREATE TABLE roles (id INTEGER PRIMARY KEY);
CREATE TABLE tenants (
  tenant_id INTEGER NOT NULL,
  id INTEGER NOT NULL,
  PRIMARY KEY (tenant_id, id)
);
CREATE TABLE posts (
  id INTEGER PRIMARY KEY,
  author_id INTEGER NOT NULL,
  tenant_id INTEGER NOT NULL,
  owner_id INTEGER NOT NULL,
  FOREIGN KEY (author_id) REFERENCES users (id),
  FOREIGN KEY (tenant_id, owner_id) REFERENCES tenants (tenant_id, id)
);
CREATE TABLE user_roles (
  user_id INTEGER NOT NULL,
  role_id INTEGER NOT NULL,
  PRIMARY KEY (user_id, role_id),
  FOREIGN KEY (user_id) REFERENCES users (id),
  FOREIGN KEY (role_id) REFERENCES roles (id)
);
CREATE TABLE friendships (
  user_id INTEGER NOT NULL,
  friend_id INTEGER NOT NULL,
  PRIMARY KEY (user_id, friend_id),
  FOREIGN KEY (user_id) REFERENCES users (id),
  FOREIGN KEY (friend_id) REFERENCES users (id)
);
CREATE INDEX posts_author ON posts (author_id);
CREATE INDEX posts_tenant_owner ON posts (tenant_id, owner_id);
CREATE INDEX user_roles_role ON user_roles (role_id);
CREATE INDEX friendships_friend ON friendships (friend_id);

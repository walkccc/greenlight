CREATE TABLE IF NOT EXISTS "Permissions" (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS "UsersPermissions" (
  user_id BIGINT NOT NULL REFERENCES "Users" ON DELETE CASCADE,
  permission_id bigint NOT NULL REFERENCES "Permissions" ON DELETE CASCADE,
  PRIMARY KEY (user_id, permission_id)
);

-- Add the two permissions to the table.
INSERT INTO "Permissions" (code)
VALUES ('movies:read'), ('movies:write');

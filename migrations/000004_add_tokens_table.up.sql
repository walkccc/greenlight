CREATE TABLE IF NOT EXISTS tokens (
  hash bytea PRIMARY KEY,
  user_id bigint NOT NULL REFERENCES users ON DELETE CASCADE,
  expiry timestamptz NOT NULL DEFAULT (now()),
  scope text NOT NULL
);

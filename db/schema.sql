CREATE TABLE IF NOT EXISTS projects (
  id SERIAL PRIMARY KEY,
  source TEXT NOT NULL,
  external_id TEXT NOT NULL,
  title TEXT NOT NULL,
  link TEXT NOT NULL,
  budget_text TEXT NOT NULL,
  amount_min BIGINT NOT NULL,
  amount_max BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_source_external ON projects (source, external_id);

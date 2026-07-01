CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS data_narratives (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  date        DATE NOT NULL,
  period_type TEXT NOT NULL CHECK (period_type IN ('daily', 'weekly')),
  content     TEXT NOT NULL,
  embedding   VECTOR(768),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(user_id, date, period_type)
);

CREATE INDEX IF NOT EXISTS idx_narratives_embedding
  ON data_narratives USING ivfflat (embedding vector_cosine_ops)
  WITH (lists = 100);

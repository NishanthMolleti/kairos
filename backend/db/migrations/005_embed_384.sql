-- Switch embedding from nomic-embed-text-v1 (768d) to all-MiniLM-L6-v2 (384d)
DROP INDEX IF EXISTS idx_narratives_embedding;
ALTER TABLE data_narratives DROP COLUMN IF EXISTS embedding;
ALTER TABLE data_narratives ADD COLUMN embedding VECTOR(384);
CREATE INDEX IF NOT EXISTS idx_narratives_embedding
  ON data_narratives USING ivfflat (embedding vector_cosine_ops)
  WITH (lists = 100);

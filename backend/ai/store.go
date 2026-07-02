package ai

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// UpsertNarrative inserts or updates the narrative and its embedding for (userID, date).
// period_type is always 'daily' for sync-generated narratives.
func UpsertNarrative(db *sqlx.DB, userID uuid.UUID, date time.Time, content string, embedding []float32) error {
	vecLiteral := float32SliceToVectorLiteral(embedding)
	_, err := db.Exec(
		`INSERT INTO data_narratives (user_id, date, period_type, content, embedding, created_at)
		 VALUES ($1, $2, 'daily', $3, $4::vector, NOW())
		 ON CONFLICT (user_id, date, period_type)
		 DO UPDATE SET content = EXCLUDED.content, embedding = EXCLUDED.embedding`,
		userID, date.Format("2006-01-02"), content, vecLiteral,
	)
	if err != nil {
		return fmt.Errorf("store: upsert narrative: %w", err)
	}
	return nil
}

// NarrativeChunk is a narrative row returned from similarity search.
type NarrativeChunk struct {
	Date       time.Time
	Content    string
	Similarity float64
}

// RetrieveSimilar returns the top `limit` narrative chunks for a user ordered by cosine similarity.
func RetrieveSimilar(db *sqlx.DB, userID uuid.UUID, embedding []float32, limit int) ([]NarrativeChunk, error) {
	vecLiteral := float32SliceToVectorLiteral(embedding)
	rows, err := db.Query(
		`SELECT date, content, 1 - (embedding <=> $2::vector) AS similarity
		 FROM data_narratives
		 WHERE user_id = $1 AND embedding IS NOT NULL
		 ORDER BY embedding <=> $2::vector
		 LIMIT $3`,
		userID, vecLiteral, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("store: retrieve similar: %w", err)
	}
	defer rows.Close()

	var chunks []NarrativeChunk
	for rows.Next() {
		var c NarrativeChunk
		if err := rows.Scan(&c.Date, &c.Content, &c.Similarity); err != nil {
			return nil, fmt.Errorf("store: scan chunk: %w", err)
		}
		chunks = append(chunks, c)
	}
	return chunks, rows.Err()
}

// GenerateAndStoreNarrative builds a narrative for userID on date, embeds it, and upserts it.
func GenerateAndStoreNarrative(db *sqlx.DB, userID uuid.UUID, date time.Time, hfAPIKey string) error {
	narrative, err := BuildNarrative(db, userID, date)
	if err != nil {
		return nil // No data — skip silently.
	}
	embedding, err := Embed(narrative, hfAPIKey)
	if err != nil {
		return fmt.Errorf("GenerateAndStoreNarrative: embed: %w", err)
	}
	return UpsertNarrative(db, userID, date, narrative, embedding)
}

// float32SliceToVectorLiteral converts []float32 to pgvector string literal "[v1,v2,...]".
func float32SliceToVectorLiteral(v []float32) string {
	sb := strings.Builder{}
	sb.WriteString("[")
	for i, f := range v {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("%g", f))
	}
	sb.WriteString("]")
	return sb.String()
}

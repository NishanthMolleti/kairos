# Kairos — Plan 2: AI Layer (Sage)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an AI layer called Sage that generates nightly health narratives from Oura data and answers natural-language questions about that data via a RAG + SQL pipeline streamed over SSE.
**Architecture:** After each Oura sync, `ai.GenerateAndStoreNarrative` builds a plain-English summary of every health metric for that date, embeds it with HuggingFace nomic-embed-text-v1 (768-dim), and stores it in `data_narratives` (pgvector). At query time, Sage generates SQL via Groq llama-3.3-70b-versatile, executes it scoped to the requesting user, retrieves the top-5 most similar narratives via cosine similarity, then streams a grounded answer back over SSE.
**Tech Stack:** Go 1.22, PostgreSQL + pgvector, HuggingFace Inference API (nomic-embed-text-v1), Groq API (llama-3.3-70b-versatile, OpenAI-compatible), Gin SSE streaming, `pgx/v5`

---

## Prerequisites

- Plan 1 is fully deployed: all models, DB tables, Oura sync, and auth middleware exist.
- `data_narratives` table has a `VECTOR(768)` embedding column (pgvector extension enabled).
- `chat_sessions` and `chat_messages` tables exist (created in Plan 1 migration).
- `cfg.GroqAPIKey` and `cfg.HuggingFaceAPIKey` are populated from environment.

---

## Task 1 — `models/chat.go`: ChatSession + ChatMessage structs and DB helpers

- [ ] Create file `Kairos/backend/models/chat.go`.

```go
package models

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ChatSession represents a conversation session between a user and Sage.
type ChatSession struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

// ChatMessage is a single turn inside a ChatSession.
type ChatMessage struct {
	ID        int64     `json:"id"`
	SessionID int64     `json:"session_id"`
	Role      string    `json:"role"` // "user" | "assistant"
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateSession inserts a new chat session and returns it with the generated ID.
func CreateSession(db *pgxpool.Pool, userID int64, title string) (*ChatSession, error) {
	s := &ChatSession{UserID: userID, Title: title}
	err := db.QueryRow(context.Background(),
		`INSERT INTO chat_sessions (user_id, title, created_at)
		 VALUES ($1, $2, NOW())
		 RETURNING id, created_at`,
		userID, title,
	).Scan(&s.ID, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// AddMessage appends a message to a session and returns it with the generated ID.
func AddMessage(db *pgxpool.Pool, sessionID int64, role, content string) (*ChatMessage, error) {
	m := &ChatMessage{SessionID: sessionID, Role: role, Content: content}
	err := db.QueryRow(context.Background(),
		`INSERT INTO chat_messages (session_id, role, content, created_at)
		 VALUES ($1, $2, $3, NOW())
		 RETURNING id, created_at`,
		sessionID, role, content,
	).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// GetSessionMessages returns all messages for a session ordered by created_at ascending.
func GetSessionMessages(db *pgxpool.Pool, sessionID int64) ([]ChatMessage, error) {
	rows, err := db.Query(context.Background(),
		`SELECT id, session_id, role, content, created_at
		 FROM chat_messages
		 WHERE session_id = $1
		 ORDER BY created_at ASC`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []ChatMessage
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}
```

- [ ] Verify `chat_sessions` and `chat_messages` tables match the column set above. If they don't exist, run this migration:

```sql
CREATE TABLE IF NOT EXISTS chat_sessions (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title      TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS chat_messages (
    id         BIGSERIAL PRIMARY KEY,
    session_id BIGINT NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    role       TEXT NOT NULL CHECK (role IN ('user','assistant')),
    content    TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## Task 2 — `ai/embed.go`: HuggingFace nomic-embed-text-v1 embedding

- [ ] Create directory `Kairos/backend/ai/` and file `Kairos/backend/ai/embed.go`.

```go
package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const hfEmbedURL = "https://api-inference.huggingface.co/pipeline/feature-extraction/nomic-ai/nomic-embed-text-v1"

type hfEmbedRequest struct {
	Inputs string `json:"inputs"`
}

// Embed calls the HuggingFace nomic-embed-text-v1 model and returns a 768-dimensional vector.
// The API returns [][]float32; we take index [0] for a single input.
func Embed(text, hfAPIKey string) ([]float32, error) {
	body, err := json.Marshal(hfEmbedRequest{Inputs: text})
	if err != nil {
		return nil, fmt.Errorf("embed: marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, hfEmbedURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("embed: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+hfAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embed: http: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("embed: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embed: HuggingFace returned %d: %s", resp.StatusCode, raw)
	}

	// The endpoint returns [][]float32 for batch inputs; we sent one string so take [0].
	var outer [][]float32
	if err := json.Unmarshal(raw, &outer); err != nil {
		return nil, fmt.Errorf("embed: unmarshal response: %w", err)
	}
	if len(outer) == 0 {
		return nil, fmt.Errorf("embed: empty response from HuggingFace")
	}
	if len(outer[0]) != 768 {
		return nil, fmt.Errorf("embed: expected 768 dims, got %d", len(outer[0]))
	}
	return outer[0], nil
}
```

---

## Task 3 — `ai/narrative.go`: BuildNarrative queries all metric tables for a date

- [ ] Create `Kairos/backend/ai/narrative.go`.

The function queries every metric table for a single `(userID, date)` pair and assembles a plain-English string. Fields that are NULL (e.g. no workout that day) are silently skipped.

```go
package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// BuildNarrative queries all health metric tables for userID on date and returns
// a descriptive string suitable for embedding and RAG retrieval.
func BuildNarrative(db *pgxpool.Pool, userID int64, date time.Time) (string, error) {
	dateStr := date.Format("January 2, 2006")
	parts := []string{fmt.Sprintf("%s:", dateStr)}

	// --- daily_sleep ---
	var sleepScore, efficiency, latency, restless *int
	var totalSleep, rem, deep, light, awake *int // seconds
	_ = db.QueryRow(context.Background(),
		`SELECT score, total_sleep_duration, efficiency, latency,
		        rem_sleep_duration, deep_sleep_duration, light_sleep_duration,
		        awake_time, restless_periods
		 FROM daily_sleep WHERE user_id=$1 AND date=$2`,
		userID, date.Format("2006-01-02"),
	).Scan(&sleepScore, &totalSleep, &efficiency, &latency,
		&rem, &deep, &light, &awake, &restless)
	if sleepScore != nil {
		parts = append(parts, fmt.Sprintf("sleep score %d", *sleepScore))
	}
	if totalSleep != nil {
		h := *totalSleep / 3600
		m := (*totalSleep % 3600) / 60
		parts = append(parts, fmt.Sprintf("total sleep %dh%dm", h, m))
	}
	if efficiency != nil {
		parts = append(parts, fmt.Sprintf("sleep efficiency %d%%", *efficiency))
	}
	if latency != nil {
		parts = append(parts, fmt.Sprintf("sleep latency %dmin", *latency/60))
	}
	if rem != nil {
		parts = append(parts, fmt.Sprintf("REM %dmin", *rem/60))
	}
	if deep != nil {
		parts = append(parts, fmt.Sprintf("deep sleep %dmin", *deep/60))
	}
	if restless != nil {
		parts = append(parts, fmt.Sprintf("restless periods %d", *restless))
	}

	// --- daily_readiness ---
	var readScore *int
	var rhr, bodyTemp, recoveryIndex, hrvBalance, activityBalance, sleepBalance *float64
	_ = db.QueryRow(context.Background(),
		`SELECT score, hrv_balance, body_temperature, recovery_index,
		        resting_heart_rate, activity_balance, sleep_balance
		 FROM daily_readiness WHERE user_id=$1 AND date=$2`,
		userID, date.Format("2006-01-02"),
	).Scan(&readScore, &hrvBalance, &bodyTemp, &recoveryIndex,
		&rhr, &activityBalance, &sleepBalance)
	if readScore != nil {
		parts = append(parts, fmt.Sprintf("readiness %d", *readScore))
	}
	if rhr != nil {
		parts = append(parts, fmt.Sprintf("resting HR %.0fbpm", *rhr))
	}
	if bodyTemp != nil {
		parts = append(parts, fmt.Sprintf("body temp deviation %.1f\u00b0C", *bodyTemp))
	}
	if recoveryIndex != nil {
		parts = append(parts, fmt.Sprintf("recovery index %.1f", *recoveryIndex))
	}

	// --- daily_activity ---
	var actScore, steps, calories, activeCalories, metMin *int
	var sedentary, lowAct, medAct, highAct *int // seconds
	_ = db.QueryRow(context.Background(),
		`SELECT score, steps, calories, active_calories, met_minutes,
		        sedentary_time, low_activity, medium_activity, high_activity
		 FROM daily_activity WHERE user_id=$1 AND date=$2`,
		userID, date.Format("2006-01-02"),
	).Scan(&actScore, &steps, &calories, &activeCalories, &metMin,
		&sedentary, &lowAct, &medAct, &highAct)
	if actScore != nil {
		parts = append(parts, fmt.Sprintf("activity score %d", *actScore))
	}
	if steps != nil {
		parts = append(parts, fmt.Sprintf("steps %d", *steps))
	}
	if calories != nil {
		parts = append(parts, fmt.Sprintf("total calories %d", *calories))
	}
	if activeCalories != nil {
		parts = append(parts, fmt.Sprintf("active calories %d", *activeCalories))
	}

	// --- daily_hrv ---
	var rmssd, bdi *float64
	_ = db.QueryRow(context.Background(),
		`SELECT rmssd, bdi FROM daily_hrv WHERE user_id=$1 AND date=$2`,
		userID, date.Format("2006-01-02"),
	).Scan(&rmssd, &bdi)
	if rmssd != nil {
		parts = append(parts, fmt.Sprintf("HRV %.0fms", *rmssd))
	}
	if bdi != nil {
		parts = append(parts, fmt.Sprintf("HRV balance %.1f", *bdi))
	}

	// --- daily_spo2 ---
	var avgSpO2, minSpO2 *float64
	_ = db.QueryRow(context.Background(),
		`SELECT avg_spo2, min_spo2 FROM daily_spo2 WHERE user_id=$1 AND date=$2`,
		userID, date.Format("2006-01-02"),
	).Scan(&avgSpO2, &minSpO2)
	if avgSpO2 != nil {
		parts = append(parts, fmt.Sprintf("avg SpO2 %.1f%%", *avgSpO2))
	}
	if minSpO2 != nil {
		parts = append(parts, fmt.Sprintf("min SpO2 %.1f%%", *minSpO2))
	}

	// --- daily_stress ---
	var stressHigh, recoveryHigh *int
	var daySummary *string
	_ = db.QueryRow(context.Background(),
		`SELECT stress_high, recovery_high, day_summary
		 FROM daily_stress WHERE user_id=$1 AND date=$2`,
		userID, date.Format("2006-01-02"),
	).Scan(&stressHigh, &recoveryHigh, &daySummary)
	if stressHigh != nil {
		parts = append(parts, fmt.Sprintf("high-stress time %dmin", *stressHigh/60))
	}
	if recoveryHigh != nil {
		parts = append(parts, fmt.Sprintf("recovery time %dmin", *recoveryHigh/60))
	}
	if daySummary != nil {
		parts = append(parts, fmt.Sprintf("day summary: %s", *daySummary))
	}

	// --- workouts (multiple per day possible) ---
	rows, err := db.Query(context.Background(),
		`SELECT activity, calories, distance,
		        EXTRACT(EPOCH FROM (end_datetime - start_datetime))::int AS duration_sec
		 FROM workouts
		 WHERE user_id=$1 AND DATE(start_datetime)=$2`,
		userID, date.Format("2006-01-02"),
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var activity *string
			var wCal, wDist *float64
			var durSec *int
			if scanErr := rows.Scan(&activity, &wCal, &wDist, &durSec); scanErr == nil {
				wParts := []string{}
				if activity != nil {
					wParts = append(wParts, *activity)
				}
				if durSec != nil {
					wParts = append(wParts, fmt.Sprintf("%dmin", *durSec/60))
				}
				if wCal != nil {
					wParts = append(wParts, fmt.Sprintf("%.0f cal", *wCal))
				}
				if wDist != nil && *wDist > 0 {
					wParts = append(wParts, fmt.Sprintf("%.2fkm", *wDist/1000))
				}
				if len(wParts) > 0 {
					parts = append(parts, "workout: "+strings.Join(wParts, " "))
				}
			}
		}
	}

	if len(parts) == 1 {
		return "", fmt.Errorf("narrative: no data found for user %d on %s", userID, date.Format("2006-01-02"))
	}
	return strings.Join(parts, ", "), nil
}
```

---

## Task 4 — `ai/store.go`: UpsertNarrative + RetrieveSimilar (pgvector)

- [ ] Create `Kairos/backend/ai/store.go`.

```go
package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UpsertNarrative inserts or updates the narrative and its embedding for (userID, date).
func UpsertNarrative(db *pgxpool.Pool, userID int64, date time.Time, content string, embedding []float32) error {
	vecLiteral := float32SliceToVectorLiteral(embedding)
	_, err := db.Exec(context.Background(),
		`INSERT INTO data_narratives (user_id, date, content, embedding, updated_at)
		 VALUES ($1, $2, $3, $4::vector, NOW())
		 ON CONFLICT (user_id, date)
		 DO UPDATE SET content = EXCLUDED.content,
		               embedding = EXCLUDED.embedding,
		               updated_at = NOW()`,
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

// RetrieveSimilar returns the top `limit` narrative chunks for a user ordered by
// cosine similarity (pgvector <=> operator = cosine distance; 1 - distance = similarity).
func RetrieveSimilar(db *pgxpool.Pool, userID int64, embedding []float32, limit int) ([]NarrativeChunk, error) {
	vecLiteral := float32SliceToVectorLiteral(embedding)
	rows, err := db.Query(context.Background(),
		`SELECT date, content, 1 - (embedding <=> $2::vector) AS similarity
		 FROM data_narratives
		 WHERE user_id = $1
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

// GenerateAndStoreNarrative is the top-level function called by oura/sync.go after each sync.
// It builds a narrative, embeds it, and upserts it into data_narratives.
func GenerateAndStoreNarrative(db *pgxpool.Pool, userID int64, date time.Time, hfAPIKey string) error {
	narrative, err := BuildNarrative(db, userID, date)
	if err != nil {
		// No data for this date — skip silently.
		return nil
	}
	embedding, err := Embed(narrative, hfAPIKey)
	if err != nil {
		return fmt.Errorf("GenerateAndStoreNarrative: embed: %w", err)
	}
	return UpsertNarrative(db, userID, date, narrative, embedding)
}

// float32SliceToVectorLiteral converts a []float32 to a pgvector string literal "[v1,v2,...]".
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
```

- [ ] Confirm `data_narratives` has a unique constraint on `(user_id, date)`. If not, add:

```sql
ALTER TABLE data_narratives
  ADD CONSTRAINT data_narratives_user_date_unique UNIQUE (user_id, date);
```

---

## Task 5 — `ai/sage.go`: full Sage pipeline (SQL gen + exec + RAG + Groq SSE stream)

- [ ] Create `Kairos/backend/ai/sage.go`.

```go
package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const groqBaseURL = "https://api.groq.com/openai/v1"
const groqModel = "llama-3.3-70b-versatile"

// ---- Groq request / response types ----

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqRequest struct {
	Model    string        `json:"model"`
	Messages []groqMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type groqStreamDelta struct {
	Content string `json:"content"`
}

type groqStreamChoice struct {
	Delta        groqStreamDelta `json:"delta"`
	FinishReason *string         `json:"finish_reason"`
}

type groqStreamChunk struct {
	Choices []groqStreamChoice `json:"choices"`
}

// ---- System prompts ----

const sqlGenSystemPrompt = `You are a SQL query generator for a health database. The user has asked a question about their health data.
Generate a single PostgreSQL SELECT query to answer the question.
Rules:
- Always include WHERE user_id = $1 (the user_id parameter will be injected)
- Only use these tables: daily_sleep, daily_readiness, daily_activity, daily_hrv, heart_rate, daily_spo2, daily_stress, workouts
- Return only the SQL query, no explanation, no markdown
- If the question cannot be answered with a SQL query, return: SELECT 'NO_SQL'

Schema:
daily_sleep(user_id, date, score, total_sleep_duration, efficiency, latency, rem_sleep_duration, deep_sleep_duration, light_sleep_duration, awake_time, restless_periods)
daily_readiness(user_id, date, score, hrv_balance, body_temperature, recovery_index, resting_heart_rate, activity_balance, sleep_balance)
daily_activity(user_id, date, score, steps, calories, active_calories, met_minutes, sedentary_time, low_activity, medium_activity, high_activity)
daily_hrv(user_id, date, rmssd, bdi)
heart_rate(user_id, timestamp, bpm, source)
daily_spo2(user_id, date, avg_spo2, min_spo2)
daily_stress(user_id, date, stress_high, recovery_high, day_summary)
workouts(user_id, start_datetime, end_datetime, activity, calories, distance)`

const sageSystemPrompt = `You are Sage, a health data assistant for Kairos. You answer questions EXCLUSIVELY based on the data provided to you.

Rules:
- Only reference facts present in the SQL results or narrative context below
- Never infer, estimate, or use external knowledge about health
- If the data does not contain enough information to answer, respond: "I don't have data to answer that."
- Be concise and direct
- When referencing numbers, always state the date they're from`

// ---- Public API ----

// Ask runs the full Sage pipeline for a user question.
// Tokens are sent to tokenCh as they arrive from Groq. The channel is closed when done.
// Streaming errors are sent as a final "ERROR:<stage>:<msg>" token before the channel closes.
func Ask(
	ctx context.Context,
	db *pgxpool.Pool,
	groqAPIKey, hfAPIKey string,
	userID int64,
	question string,
	tokenCh chan<- string,
) {
	defer close(tokenCh)

	// Step 1 — Generate SQL.
	sqlQuery, err := generateSQL(ctx, groqAPIKey, question)
	if err != nil {
		tokenCh <- fmt.Sprintf("ERROR:sql_gen:%v", err)
		return
	}

	// Step 2 — Execute SQL scoped to user.
	sqlResults := ""
	noSQL := strings.EqualFold(strings.TrimSpace(sqlQuery), "select 'no_sql'")
	if !noSQL {
		sqlResults, err = executeSQL(ctx, db, userID, sqlQuery)
		if err != nil {
			// Non-fatal — Sage will answer from narrative context only.
			sqlResults = fmt.Sprintf("[SQL execution error: %v]", err)
		}
	}

	// Step 3 — Embed question + RAG retrieval.
	narrativeContext := ""
	embedding, embedErr := Embed(question, hfAPIKey)
	if embedErr == nil {
		chunks, rerr := RetrieveSimilar(db, userID, embedding, 5)
		if rerr == nil && len(chunks) > 0 {
			var sb strings.Builder
			for _, c := range chunks {
				sb.WriteString(c.Date.Format("2006-01-02"))
				sb.WriteString(": ")
				sb.WriteString(c.Content)
				sb.WriteString("\n")
			}
			narrativeContext = sb.String()
		}
	}

	// Step 4 — Build the user message with injected context.
	userMsg := fmt.Sprintf("Question: %s\n\nSQL Results:\n%s\n\nNarrative Context:\n%s",
		question, sqlResults, narrativeContext)

	// Step 5 — Stream Groq answer.
	if err := streamGroqAnswer(ctx, groqAPIKey, userMsg, tokenCh); err != nil {
		tokenCh <- fmt.Sprintf("ERROR:stream:%v", err)
	}
}

// ---- Internal helpers ----

func generateSQL(ctx context.Context, groqAPIKey, question string) (string, error) {
	reqBody := groqRequest{
		Model: groqModel,
		Messages: []groqMessage{
			{Role: "system", Content: sqlGenSystemPrompt},
			{Role: "user", Content: question},
		},
		Stream: false,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal groq sql request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		groqBaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+groqAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("groq sql gen http: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("groq sql gen %d: %s", resp.StatusCode, raw)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("groq sql gen unmarshal: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("groq sql gen: empty choices")
	}
	sql := strings.TrimSpace(result.Choices[0].Message.Content)
	// Strip accidental markdown code fences.
	sql = strings.TrimPrefix(sql, "```sql")
	sql = strings.TrimPrefix(sql, "```")
	sql = strings.TrimSuffix(sql, "```")
	return strings.TrimSpace(sql), nil
}

// executeSQL runs the generated SQL with userID bound to $1 and returns a plain-text table.
func executeSQL(ctx context.Context, db *pgxpool.Pool, userID int64, sqlQuery string) (string, error) {
	rows, err := db.Query(ctx, sqlQuery, userID)
	if err != nil {
		return "", fmt.Errorf("execute: %w", err)
	}
	defer rows.Close()

	fieldDescs := rows.FieldDescriptions()
	headers := make([]string, len(fieldDescs))
	for i, fd := range fieldDescs {
		headers[i] = string(fd.Name)
	}

	var sb strings.Builder
	sb.WriteString(strings.Join(headers, " | "))
	sb.WriteString("\n")

	rowCount := 0
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return "", fmt.Errorf("execute scan: %w", err)
		}
		strs := make([]string, len(vals))
		for i, v := range vals {
			strs[i] = fmt.Sprintf("%v", v)
		}
		sb.WriteString(strings.Join(strs, " | "))
		sb.WriteString("\n")
		rowCount++
		if rowCount >= 100 { // Safety cap — prevent enormous context payloads.
			sb.WriteString("... (truncated at 100 rows)\n")
			break
		}
	}
	if rowCount == 0 {
		sb.WriteString("(no rows)\n")
	}
	return sb.String(), rows.Err()
}

// streamGroqAnswer sends the Sage system prompt + userMsg to Groq with stream:true
// and forwards each token to tokenCh.
func streamGroqAnswer(ctx context.Context, groqAPIKey, userMsg string, tokenCh chan<- string) error {
	reqBody := groqRequest{
		Model: groqModel,
		Messages: []groqMessage{
			{Role: "system", Content: sageSystemPrompt},
			{Role: "user", Content: userMsg},
		},
		Stream: true,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal stream request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		groqBaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+groqAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("groq stream http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("groq stream %d: %s", resp.StatusCode, raw)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var chunk groqStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // Ignore malformed lines.
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				select {
				case tokenCh <- choice.Delta.Content:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}
	return scanner.Err()
}
```

---

## Task 6 — `handlers/chat.go`: REST endpoints + SSE streaming

- [ ] Create `Kairos/backend/handlers/chat.go`.

```go
package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/kairos/ai"
	"github.com/your-org/kairos/config"
	"github.com/your-org/kairos/models"
)

// CreateSession handles POST /api/chat/sessions
func CreateSession(db *pgxpool.Pool, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("userID")
		var body struct {
			Title string `json:"title"`
		}
		_ = c.ShouldBindJSON(&body)
		if body.Title == "" {
			body.Title = "New conversation"
		}
		session, err := models.CreateSession(db, userID, body.Title)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, session)
	}
}

// GetSessionMessages handles GET /api/chat/sessions/:id/messages
func GetSessionMessages(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("userID")
		sessionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
			return
		}
		ownerID, err := sessionOwner(c.Request.Context(), db, sessionID)
		if err != nil || ownerID != userID {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		msgs, err := models.GetSessionMessages(db, sessionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, msgs)
	}
}

// AskSage handles POST /api/chat/sessions/:id/ask
// It streams the Sage response as Server-Sent Events (SSE).
func AskSage(db *pgxpool.Pool, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("userID")
		sessionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
			return
		}
		var body struct {
			Question string `json:"question" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "question is required"})
			return
		}
		ownerID, err := sessionOwner(c.Request.Context(), db, sessionID)
		if err != nil || ownerID != userID {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}

		// Persist the user message before streaming begins.
		if _, err := models.AddMessage(db, sessionID, "user", body.Question); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Set SSE headers.
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no") // Disable Nginx buffering.

		tokenCh := make(chan string, 64)
		ctx := c.Request.Context()
		go ai.Ask(ctx, db, cfg.GroqAPIKey, cfg.HuggingFaceAPIKey, userID, body.Question, tokenCh)

		var fullResponse strings.Builder
		w := c.Writer
		flusher, hasFlusher := w.(http.Flusher)

		for token := range tokenCh {
			if strings.HasPrefix(token, "ERROR:") {
				fmt.Fprintf(w, "event: error\ndata: %s\n\n", token)
				if hasFlusher {
					flusher.Flush()
				}
				break
			}
			fullResponse.WriteString(token)
			// Escape embedded newlines so they don't break SSE framing.
			escaped := strings.ReplaceAll(token, "\n", "\ndata: ")
			fmt.Fprintf(w, "data: %s\n\n", escaped)
			if hasFlusher {
				flusher.Flush()
			}
		}

		// Send terminal done event.
		fmt.Fprintf(w, "event: done\ndata: [DONE]\n\n")
		if hasFlusher {
			flusher.Flush()
		}

		// Persist the full assistant response to chat_messages.
		if fullResponse.Len() > 0 {
			_, _ = models.AddMessage(db, sessionID, "assistant", fullResponse.String())
		}
	}
}

// sessionOwner looks up the user_id that owns a chat session.
func sessionOwner(ctx context.Context, db *pgxpool.Pool, sessionID int64) (int64, error) {
	var ownerID int64
	err := db.QueryRow(ctx,
		`SELECT user_id FROM chat_sessions WHERE id = $1`, sessionID,
	).Scan(&ownerID)
	return ownerID, err
}
```

---

## Task 7 — Wire chat handlers into `main.go`

- [ ] Open `Kairos/backend/main.go`.
- [ ] Inside the authenticated `api` router group, after the last existing route registration, add exactly these 3 lines:

```diff
 api.GET("/sync", handlers.SyncNow(db, cfg))
+api.POST("/chat/sessions", handlers.CreateSession(db, cfg))
+api.GET("/chat/sessions/:id/messages", handlers.GetSessionMessages(db))
+api.POST("/chat/sessions/:id/ask", handlers.AskSage(db, cfg))
 
 router.Run(":" + cfg.Port)
```

No other changes to `main.go`.

---

## Task 8 — Modify `oura/sync.go` to call narrative generation after sync

- [ ] Open `Kairos/backend/oura/sync.go`.
- [ ] Add `"github.com/your-org/kairos/ai"` and `"log"` to the import block (if not already present).
- [ ] Inside `SyncUser`, after all 9 endpoint sync calls complete for a given date, add:

```go
// After all endpoint sync calls for syncDate complete:
if err := ai.GenerateAndStoreNarrative(db, userID, syncDate, cfg.HuggingFaceAPIKey); err != nil {
    // Log but do not fail the sync — narrative generation is best-effort.
    log.Printf("narrative generation failed for user %d on %s: %v",
        userID, syncDate.Format("2006-01-02"), err)
}
```

If `SyncUser` iterates over a date range, place this call at the bottom of each iteration, still inside the loop.

---

## Task 9 — Smoke tests

Run against the local server. `$TOKEN` is a valid JWT; server on `:8080`.

### 9.1 — Create a chat session

```bash
curl -s -X POST http://localhost:8080/api/chat/sessions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Test session"}' | jq .
# Expected: {"id":1,"user_id":...,"title":"Test session","created_at":"..."}
```

### 9.2 — Ask Sage a question (SSE stream)

```bash
curl -s -N -X POST http://localhost:8080/api/chat/sessions/1/ask \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"question":"What was my sleep score last night?"}'
# Expected: stream of `data: <token>` lines ending with `event: done\ndata: [DONE]`
```

### 9.3 — Fetch session history

```bash
curl -s http://localhost:8080/api/chat/sessions/1/messages \
  -H "Authorization: Bearer $TOKEN" | jq .
# Expected: JSON array with two messages — role "user" and role "assistant"
```

### 9.4 — Trigger Oura sync (exercises narrative generation path)

```bash
curl -s http://localhost:8080/api/sync \
  -H "Authorization: Bearer $TOKEN" | jq .
# Expected: {"status":"ok"} — check server logs for "narrative generation" lines
```

### 9.5 — Verify narrative was stored (Postgres)

```sql
SELECT user_id, date, LEFT(content, 80) AS preview,
       array_length(embedding::float4[], 1) AS dims
FROM data_narratives
ORDER BY updated_at DESC
LIMIT 5;
-- Expected: rows with 768-dim embeddings and readable health summaries.
```

### 9.6 — Ownership guard (expect 404)

```bash
curl -s http://localhost:8080/api/chat/sessions/9999/messages \
  -H "Authorization: Bearer $TOKEN" | jq .
# Expected: {"error":"session not found"}
```

---

## Self-Review Checklist

- [ ] **Spec coverage:** All 6 new files (`models/chat.go`, `ai/embed.go`, `ai/narrative.go`, `ai/store.go`, `ai/sage.go`, `handlers/chat.go`) are fully implemented with no placeholder code.
- [ ] **No placeholders:** Zero "TBD", "TODO", or stub implementations. Every function body runs to completion.
- [ ] **Type consistency:** `userID int64` threaded through all models, AI, and handler layers. `[]float32` used for embeddings end-to-end. `*pgxpool.Pool` used for all DB access.
- [ ] **Pipeline completeness:** Sage `Ask` covers all 5 steps: SQL gen → exec → embed → RAG → Groq stream.
- [ ] **Security:** Every `executeSQL` call binds `userID` as `$1`. Session ownership checked before every session-scoped operation via `sessionOwner`.
- [ ] **Error handling:** `GenerateAndStoreNarrative` returns nil on "no data". SQL execution errors degrade gracefully to narrative-only mode. Streaming errors signalled via `ERROR:` prefix token.
- [ ] **SSE correctness:** `Content-Type: text/event-stream`, `X-Accel-Buffering: no`, `http.Flusher` used, `event: done` sent at stream end.
- [ ] **Persistence:** User turn persisted before streaming; full assistant response persisted after channel drains.
- [ ] **oura/sync.go change:** `ai.GenerateAndStoreNarrative` called after each date sync; failure is logged, not propagated.
- [ ] **main.go wiring:** Exactly 3 new route lines added, no other changes.
- [ ] **pgvector literal:** `float32SliceToVectorLiteral` produces `[v1,v2,...]` accepted by pgvector via `::vector` cast.
- [ ] **HuggingFace response shape:** `[][]float32` outer array; `[0]` taken for single input; 768-dim assertion present.
- [ ] **Groq streaming:** `[DONE]` sentinel handled; malformed SSE lines skipped; context cancellation respected via `select`.

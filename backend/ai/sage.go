package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

const groqBaseURL = "https://api.groq.com/openai/v1"
const groqModel = "llama-3.3-70b-versatile"

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

const sqlGenSystemPrompt = `You are a SQL query generator for a health database. Generate a single PostgreSQL SELECT query to answer the user's question.
Rules:
- Always include WHERE user_id = $1
- Only use these tables: daily_sleep, daily_readiness, daily_activity, daily_hrv, heart_rate, daily_spo2, daily_stress, workouts
- Return ONLY the SQL query, no explanation, no markdown fences
- If the question cannot be answered with SQL, return exactly: SELECT 'NO_SQL'

Schema:
daily_sleep(user_id UUID, date DATE, score INT, total_sleep_duration INT, efficiency INT, latency INT, rem_sleep_duration INT, deep_sleep_duration INT, light_sleep_duration INT, awake_time INT, restless_periods INT)
daily_readiness(user_id UUID, date DATE, score INT, hrv_balance INT, body_temperature FLOAT, recovery_index INT, resting_heart_rate INT, activity_balance INT, sleep_balance INT)
daily_activity(user_id UUID, date DATE, score INT, steps INT, calories INT, active_calories INT, met_minutes FLOAT, sedentary_time INT, low_activity INT, medium_activity INT, high_activity INT)
daily_hrv(user_id UUID, date DATE, rmssd FLOAT, bdi FLOAT)
heart_rate(user_id UUID, timestamp TIMESTAMPTZ, bpm INT, source TEXT)
daily_spo2(user_id UUID, date DATE, avg_spo2 FLOAT, min_spo2 FLOAT)
daily_stress(user_id UUID, date DATE, stress_high INT, recovery_high INT, day_summary TEXT)
workouts(user_id UUID, start_datetime TIMESTAMPTZ, end_datetime TIMESTAMPTZ, activity TEXT, calories INT, distance FLOAT)`

const sageSystemPrompt = `You are Sage, a health data assistant for Kairos. Answer questions EXCLUSIVELY from the data provided.

Rules:
- Only reference facts present in the SQL results or narrative context below
- Never infer, estimate, or use external health knowledge
- If the data does not contain enough information to answer, respond: "I don't have data to answer that."
- Be concise and direct
- When referencing numbers, always state the date they are from`

// Ask runs the full Sage pipeline for a user question.
// Tokens stream to tokenCh; channel is closed when done.
// Errors are sent as "ERROR:<stage>:<msg>" before channel close.
func Ask(
	ctx context.Context,
	db *sqlx.DB,
	groqAPIKey, hfAPIKey string,
	userID uuid.UUID,
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
	log.Printf("sage sql_gen: %q", sqlQuery)

	// Step 2 — Execute SQL scoped to user.
	sqlResults := ""
	noSQL := strings.EqualFold(strings.TrimSpace(sqlQuery), "select 'no_sql'")
	if !noSQL {
		sqlResults, err = executeSQL(ctx, db, userID, sqlQuery)
		if err != nil {
			sqlResults = fmt.Sprintf("[SQL execution error: %v]", err)
		}
	}
	log.Printf("sage sql_results: noSQL=%v results=%q", noSQL, sqlResults)

	// Step 3 — Embed question + RAG retrieval.
	narrativeContext := ""
	embedding, embedErr := Embed(question, hfAPIKey)
	if embedErr != nil {
		log.Printf("sage embed error: %v", embedErr)
	} else {
		chunks, rerr := RetrieveSimilar(db, userID, embedding, 5)
		if rerr != nil {
			log.Printf("sage retrieve error: %v", rerr)
		} else {
			log.Printf("sage retrieved %d chunks", len(chunks))
			if len(chunks) > 0 {
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
	}

	// Step 4 — Build user message with injected context.
	userMsg := fmt.Sprintf("Question: %s\n\nSQL Results:\n%s\n\nNarrative Context:\n%s",
		question, sqlResults, narrativeContext)

	// Step 5 — Stream Groq answer.
	if err := streamGroqAnswer(ctx, groqAPIKey, userMsg, tokenCh); err != nil {
		tokenCh <- fmt.Sprintf("ERROR:stream:%v", err)
	}
}

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
	sql = strings.TrimPrefix(sql, "```sql")
	sql = strings.TrimPrefix(sql, "```")
	sql = strings.TrimSuffix(sql, "```")
	return strings.TrimSpace(sql), nil
}

// executeSQL runs the Groq-generated SQL with userID bound to $1 and returns a plain-text result table.
func executeSQL(ctx context.Context, db *sqlx.DB, userID uuid.UUID, sqlQuery string) (string, error) {
	rows, err := db.QueryContext(ctx, sqlQuery, userID)
	if err != nil {
		return "", fmt.Errorf("execute: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("execute columns: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(strings.Join(cols, " | "))
	sb.WriteString("\n")

	rowCount := 0
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return "", fmt.Errorf("execute scan: %w", err)
		}
		strs := make([]string, len(vals))
		for i, v := range vals {
			strs[i] = fmt.Sprintf("%v", v)
		}
		sb.WriteString(strings.Join(strs, " | "))
		sb.WriteString("\n")
		rowCount++
		if rowCount >= 100 {
			sb.WriteString("... (truncated at 100 rows)\n")
			break
		}
	}
	if rowCount == 0 {
		sb.WriteString("(no rows)\n")
	}
	return sb.String(), rows.Err()
}

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
			continue
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

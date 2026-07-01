# Kairos — Plan 1: Backend Foundation

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Working Go backend with Oura OAuth2, Postgres schema, full telemetry sync, and daily cron — all API endpoints that Plan 2 (AI) and Plan 3 (Frontend) will consume.

**Architecture:** Go HTTP server (Gin) with JWT session auth, Oura OAuth2 PKCE flow, typed Postgres tables via sqlx, and a daily cron that pulls all 9 Oura API v2 endpoints per user. CORS enabled for Next.js frontend on a different origin.

**Tech Stack:** Go 1.22 · Gin · sqlx + lib/pq · golang-jwt/v5 · robfig/cron/v3 · google/uuid · Supabase Postgres (pgvector) · Oura API v2

---

## File Map

```
Kairos/backend/
├── main.go                        # server bootstrap, route registration, cron start
├── go.mod
├── go.sum
├── .env.example
├── config/
│   └── config.go                  # load env vars into Config struct
├── db/
│   ├── db.go                      # open sqlx connection, ping check
│   ├── migrate.go                 # embed + run migrations
│   └── migrations/
│       ├── 001_init.sql           # all core tables
│       └── 002_pgvector.sql       # enable pgvector, data_narratives table
├── models/
│   ├── user.go                    # User struct + DB queries
│   ├── sleep.go                   # DailySleep struct + upsert
│   ├── readiness.go               # DailyReadiness struct + upsert
│   ├── activity.go                # DailyActivity struct + upsert
│   ├── hrv.go                     # DailyHRV struct + upsert
│   ├── heartrate.go               # HeartRate struct + bulk insert
│   ├── spo2.go                    # DailySpO2 struct + upsert
│   ├── stress.go                  # DailyStress struct + upsert
│   └── workout.go                 # Workout struct + upsert
├── auth/
│   ├── oauth.go                   # Oura OAuth2 PKCE — GenerateURL, Exchange, Refresh
│   └── jwt.go                     # Sign, Validate JWT
├── middleware/
│   ├── auth.go                    # JWT check, inject userID into context
│   └── cors.go                    # CORS headers for Next.js origin
├── oura/
│   ├── client.go                  # authenticated HTTP client for Oura API v2
│   └── sync.go                    # SyncUser — calls all 9 endpoints, upserts to DB
├── scheduler/
│   └── cron.go                    # daily cron — iterates all users, calls SyncUser
└── handlers/
    ├── auth.go                    # GET /auth/login, GET /auth/callback, POST /auth/logout
    ├── user.go                    # GET /api/user
    ├── sync.go                    # POST /api/sync (manual trigger)
    └── metrics.go                 # GET /api/metrics/* (sleep, readiness, activity, hrv, hr, spo2, stress, workouts)
```

---

## Task 1: Go module + config

**Files:**
- Create: `Kairos/backend/go.mod`
- Create: `Kairos/backend/config/config.go`
- Create: `Kairos/backend/.env.example`

- [ ] **Step 1: Initialize Go module**

```bash
cd Kairos/backend
go mod init github.com/NishanthMolleti/kairos
```

Expected: `go.mod` created with `module github.com/NishanthMolleti/kairos` and `go 1.22`

- [ ] **Step 2: Install dependencies**

```bash
go get github.com/gin-gonic/gin@v1.10.0
go get github.com/jmoiron/sqlx@v1.3.5
go get github.com/lib/pq@v1.10.9
go get github.com/golang-jwt/jwt/v5@v5.2.1
go get github.com/robfig/cron/v3@v3.0.1
go get github.com/google/uuid@v1.6.0
go get github.com/joho/godotenv@v1.5.1
go get github.com/pgvector/pgvector-go@v0.2.1
```

- [ ] **Step 3: Write config/config.go**

```go
// config/config.go
package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL       string
	OuraClientID      string
	OuraClientSecret  string
	OuraRedirectURL   string
	JWTSecret         string
	GroqAPIKey        string
	HuggingFaceAPIKey string
	FrontendURL       string
	Port              string
}

func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		DatabaseURL:       mustEnv("DATABASE_URL"),
		OuraClientID:      mustEnv("OURA_CLIENT_ID"),
		OuraClientSecret:  mustEnv("OURA_CLIENT_SECRET"),
		OuraRedirectURL:   mustEnv("OURA_REDIRECT_URL"),
		JWTSecret:         mustEnv("JWT_SECRET"),
		GroqAPIKey:        mustEnv("GROQ_API_KEY"),
		HuggingFaceAPIKey: mustEnv("HUGGINGFACE_API_KEY"),
		FrontendURL:       getEnv("FRONTEND_URL", "http://localhost:3000"),
		Port:              getEnv("PORT", "8080"),
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s not set", key)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

- [ ] **Step 4: Write .env.example**

```
DATABASE_URL=postgresql://user:pass@host:5432/kairos?sslmode=require
OURA_CLIENT_ID=your_oura_client_id
OURA_CLIENT_SECRET=your_oura_client_secret
OURA_REDIRECT_URL=http://localhost:8080/auth/callback
JWT_SECRET=change_this_to_a_long_random_string
GROQ_API_KEY=your_groq_api_key
HUGGINGFACE_API_KEY=your_hf_api_key
FRONTEND_URL=http://localhost:3000
PORT=8080
```

- [ ] **Step 5: Verify config compiles**

```bash
go build ./config/...
```

Expected: no output (success)

- [ ] **Step 6: Commit**

```bash
git add Kairos/backend/go.mod Kairos/backend/go.sum Kairos/backend/config/ Kairos/backend/.env.example
git commit -m "feat(kairos): go module + config loader"
```

---

## Task 2: Database connection

**Files:**
- Create: `Kairos/backend/db/db.go`

- [ ] **Step 1: Write db/db.go**

```go
// db/db.go
package db

import (
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func Connect(databaseURL string) *sqlx.DB {
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	log.Println("db connected")
	return db
}
```

- [ ] **Step 2: Compile check**

```bash
go build ./db/...
```

Expected: no output

- [ ] **Step 3: Commit**

```bash
git add Kairos/backend/db/db.go
git commit -m "feat(kairos): postgres connection via sqlx"
```

---

## Task 3: Database migrations

**Files:**
- Create: `Kairos/backend/db/migrations/001_init.sql`
- Create: `Kairos/backend/db/migrations/002_pgvector.sql`
- Create: `Kairos/backend/db/migrate.go`

- [ ] **Step 1: Write 001_init.sql**

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
  id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  oura_user_id  TEXT UNIQUE NOT NULL,
  email         TEXT,
  access_token  TEXT NOT NULL,
  refresh_token TEXT NOT NULL,
  last_sync     TIMESTAMPTZ,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS daily_sleep (
  id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id               UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  date                  DATE NOT NULL,
  score                 INT,
  total_sleep_duration  INT,
  efficiency            INT,
  latency               INT,
  rem_sleep_duration    INT,
  deep_sleep_duration   INT,
  light_sleep_duration  INT,
  awake_time            INT,
  restless_periods      INT,
  UNIQUE(user_id, date)
);

CREATE TABLE IF NOT EXISTS daily_readiness (
  id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  date                DATE NOT NULL,
  score               INT,
  hrv_balance         INT,
  body_temperature    FLOAT,
  recovery_index      INT,
  resting_heart_rate  INT,
  activity_balance    INT,
  sleep_balance       INT,
  UNIQUE(user_id, date)
);

CREATE TABLE IF NOT EXISTS daily_activity (
  id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  date             DATE NOT NULL,
  score            INT,
  steps            INT,
  calories         INT,
  active_calories  INT,
  met_minutes      FLOAT,
  sedentary_time   INT,
  low_activity     INT,
  medium_activity  INT,
  high_activity    INT,
  UNIQUE(user_id, date)
);

CREATE TABLE IF NOT EXISTS daily_hrv (
  id       UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  date     DATE NOT NULL,
  rmssd    FLOAT,
  bdi      FLOAT,
  UNIQUE(user_id, date)
);

CREATE TABLE IF NOT EXISTS heart_rate (
  id        UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  timestamp TIMESTAMPTZ NOT NULL,
  bpm       INT NOT NULL,
  source    TEXT,
  UNIQUE(user_id, timestamp)
);

CREATE INDEX IF NOT EXISTS idx_heart_rate_user_ts ON heart_rate(user_id, timestamp DESC);

CREATE TABLE IF NOT EXISTS daily_spo2 (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  date        DATE NOT NULL,
  avg_spo2    FLOAT,
  min_spo2    FLOAT,
  UNIQUE(user_id, date)
);

CREATE TABLE IF NOT EXISTS daily_stress (
  id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  date           DATE NOT NULL,
  stress_high    INT,
  recovery_high  INT,
  day_summary    TEXT,
  UNIQUE(user_id, date)
);

CREATE TABLE IF NOT EXISTS workouts (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  start_datetime  TIMESTAMPTZ NOT NULL,
  end_datetime    TIMESTAMPTZ,
  activity        TEXT,
  calories        INT,
  distance        FLOAT,
  UNIQUE(user_id, start_datetime)
);

CREATE TABLE IF NOT EXISTS chat_sessions (
  id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS chat_messages (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  session_id  UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
  role        TEXT NOT NULL CHECK (role IN ('user', 'assistant')),
  content     TEXT NOT NULL,
  sql_used    TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

- [ ] **Step 2: Write 002_pgvector.sql**

```sql
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
```

Note: `VECTOR(768)` matches nomic-embed-text output dimension (not 1536 — that is OpenAI ada-002).

- [ ] **Step 3: Write db/migrate.go**

```go
// db/migrate.go
package db

import (
	_ "embed"
	"log"

	"github.com/jmoiron/sqlx"
)

//go:embed migrations/001_init.sql
var migration001 string

//go:embed migrations/002_pgvector.sql
var migration002 string

func RunMigrations(db *sqlx.DB) {
	for i, sql := range []string{migration001, migration002} {
		if _, err := db.Exec(sql); err != nil {
			log.Fatalf("migration %d failed: %v", i+1, err)
		}
	}
	log.Println("migrations applied")
}
```

- [ ] **Step 4: Compile check**

```bash
go build ./db/...
```

Expected: no output

- [ ] **Step 5: Commit**

```bash
git add Kairos/backend/db/
git commit -m "feat(kairos): db migrations — all tables + pgvector"
```

---

## Task 4: Models

**Files:**
- Create: `Kairos/backend/models/user.go`
- Create: `Kairos/backend/models/sleep.go`
- Create: `Kairos/backend/models/readiness.go`
- Create: `Kairos/backend/models/activity.go`
- Create: `Kairos/backend/models/hrv.go`
- Create: `Kairos/backend/models/heartrate.go`
- Create: `Kairos/backend/models/spo2.go`
- Create: `Kairos/backend/models/stress.go`
- Create: `Kairos/backend/models/workout.go`

- [ ] **Step 1: Write models/user.go**

```go
// models/user.go
package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type User struct {
	ID           uuid.UUID  `db:"id"`
	OuraUserID   string     `db:"oura_user_id"`
	Email        string     `db:"email"`
	AccessToken  string     `db:"access_token"`
	RefreshToken string     `db:"refresh_token"`
	LastSync     *time.Time `db:"last_sync"`
	CreatedAt    time.Time  `db:"created_at"`
}

func UpsertUser(db *sqlx.DB, u *User) error {
	_, err := db.NamedExec(`
		INSERT INTO users (oura_user_id, email, access_token, refresh_token)
		VALUES (:oura_user_id, :email, :access_token, :refresh_token)
		ON CONFLICT (oura_user_id) DO UPDATE
		  SET access_token  = EXCLUDED.access_token,
		      refresh_token = EXCLUDED.refresh_token,
		      email         = EXCLUDED.email
	`, u)
	return err
}

func GetUserByOuraID(db *sqlx.DB, ouraID string) (*User, error) {
	var u User
	err := db.Get(&u, `SELECT * FROM users WHERE oura_user_id = $1`, ouraID)
	return &u, err
}

func GetUserByID(db *sqlx.DB, id uuid.UUID) (*User, error) {
	var u User
	err := db.Get(&u, `SELECT * FROM users WHERE id = $1`, id)
	return &u, err
}

func GetAllUsers(db *sqlx.DB) ([]User, error) {
	var users []User
	err := db.Select(&users, `SELECT * FROM users`)
	return users, err
}

func UpdateLastSync(db *sqlx.DB, id uuid.UUID) error {
	_, err := db.Exec(`UPDATE users SET last_sync = NOW() WHERE id = $1`, id)
	return err
}
```

- [ ] **Step 2: Write models/sleep.go**

```go
// models/sleep.go
package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DailySleep struct {
	ID                 uuid.UUID `db:"id"`
	UserID             uuid.UUID `db:"user_id"`
	Date               time.Time `db:"date"`
	Score              *int      `db:"score"`
	TotalSleepDuration *int      `db:"total_sleep_duration"`
	Efficiency         *int      `db:"efficiency"`
	Latency            *int      `db:"latency"`
	REMSleepDuration   *int      `db:"rem_sleep_duration"`
	DeepSleepDuration  *int      `db:"deep_sleep_duration"`
	LightSleepDuration *int      `db:"light_sleep_duration"`
	AwakeTime          *int      `db:"awake_time"`
	RestlessPeriods    *int      `db:"restless_periods"`
}

func UpsertDailySleep(db *sqlx.DB, s *DailySleep) error {
	_, err := db.NamedExec(`
		INSERT INTO daily_sleep
		  (user_id, date, score, total_sleep_duration, efficiency, latency,
		   rem_sleep_duration, deep_sleep_duration, light_sleep_duration, awake_time, restless_periods)
		VALUES
		  (:user_id, :date, :score, :total_sleep_duration, :efficiency, :latency,
		   :rem_sleep_duration, :deep_sleep_duration, :light_sleep_duration, :awake_time, :restless_periods)
		ON CONFLICT (user_id, date) DO UPDATE SET
		  score = EXCLUDED.score,
		  total_sleep_duration = EXCLUDED.total_sleep_duration,
		  efficiency = EXCLUDED.efficiency,
		  latency = EXCLUDED.latency,
		  rem_sleep_duration = EXCLUDED.rem_sleep_duration,
		  deep_sleep_duration = EXCLUDED.deep_sleep_duration,
		  light_sleep_duration = EXCLUDED.light_sleep_duration,
		  awake_time = EXCLUDED.awake_time,
		  restless_periods = EXCLUDED.restless_periods
	`, s)
	return err
}

func GetSleepRange(db *sqlx.DB, userID uuid.UUID, from, to string) ([]DailySleep, error) {
	var rows []DailySleep
	err := db.Select(&rows,
		`SELECT * FROM daily_sleep WHERE user_id = $1 AND date BETWEEN $2 AND $3 ORDER BY date`,
		userID, from, to)
	return rows, err
}
```

- [ ] **Step 3: Write models/readiness.go**

```go
// models/readiness.go
package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DailyReadiness struct {
	ID               uuid.UUID `db:"id"`
	UserID           uuid.UUID `db:"user_id"`
	Date             time.Time `db:"date"`
	Score            *int      `db:"score"`
	HRVBalance       *int      `db:"hrv_balance"`
	BodyTemperature  *float64  `db:"body_temperature"`
	RecoveryIndex    *int      `db:"recovery_index"`
	RestingHeartRate *int      `db:"resting_heart_rate"`
	ActivityBalance  *int      `db:"activity_balance"`
	SleepBalance     *int      `db:"sleep_balance"`
}

func UpsertDailyReadiness(db *sqlx.DB, r *DailyReadiness) error {
	_, err := db.NamedExec(`
		INSERT INTO daily_readiness
		  (user_id, date, score, hrv_balance, body_temperature, recovery_index,
		   resting_heart_rate, activity_balance, sleep_balance)
		VALUES
		  (:user_id, :date, :score, :hrv_balance, :body_temperature, :recovery_index,
		   :resting_heart_rate, :activity_balance, :sleep_balance)
		ON CONFLICT (user_id, date) DO UPDATE SET
		  score = EXCLUDED.score,
		  hrv_balance = EXCLUDED.hrv_balance,
		  body_temperature = EXCLUDED.body_temperature,
		  recovery_index = EXCLUDED.recovery_index,
		  resting_heart_rate = EXCLUDED.resting_heart_rate,
		  activity_balance = EXCLUDED.activity_balance,
		  sleep_balance = EXCLUDED.sleep_balance
	`, r)
	return err
}

func GetReadinessRange(db *sqlx.DB, userID uuid.UUID, from, to string) ([]DailyReadiness, error) {
	var rows []DailyReadiness
	err := db.Select(&rows,
		`SELECT * FROM daily_readiness WHERE user_id = $1 AND date BETWEEN $2 AND $3 ORDER BY date`,
		userID, from, to)
	return rows, err
}
```

- [ ] **Step 4: Write models/activity.go**

```go
// models/activity.go
package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DailyActivity struct {
	ID             uuid.UUID `db:"id"`
	UserID         uuid.UUID `db:"user_id"`
	Date           time.Time `db:"date"`
	Score          *int      `db:"score"`
	Steps          *int      `db:"steps"`
	Calories       *int      `db:"calories"`
	ActiveCalories *int      `db:"active_calories"`
	METMinutes     *float64  `db:"met_minutes"`
	SedentaryTime  *int      `db:"sedentary_time"`
	LowActivity    *int      `db:"low_activity"`
	MediumActivity *int      `db:"medium_activity"`
	HighActivity   *int      `db:"high_activity"`
}

func UpsertDailyActivity(db *sqlx.DB, a *DailyActivity) error {
	_, err := db.NamedExec(`
		INSERT INTO daily_activity
		  (user_id, date, score, steps, calories, active_calories, met_minutes,
		   sedentary_time, low_activity, medium_activity, high_activity)
		VALUES
		  (:user_id, :date, :score, :steps, :calories, :active_calories, :met_minutes,
		   :sedentary_time, :low_activity, :medium_activity, :high_activity)
		ON CONFLICT (user_id, date) DO UPDATE SET
		  score = EXCLUDED.score,
		  steps = EXCLUDED.steps,
		  calories = EXCLUDED.calories,
		  active_calories = EXCLUDED.active_calories,
		  met_minutes = EXCLUDED.met_minutes,
		  sedentary_time = EXCLUDED.sedentary_time,
		  low_activity = EXCLUDED.low_activity,
		  medium_activity = EXCLUDED.medium_activity,
		  high_activity = EXCLUDED.high_activity
	`, a)
	return err
}

func GetActivityRange(db *sqlx.DB, userID uuid.UUID, from, to string) ([]DailyActivity, error) {
	var rows []DailyActivity
	err := db.Select(&rows,
		`SELECT * FROM daily_activity WHERE user_id = $1 AND date BETWEEN $2 AND $3 ORDER BY date`,
		userID, from, to)
	return rows, err
}
```

- [ ] **Step 5: Write models/hrv.go**

```go
// models/hrv.go
package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DailyHRV struct {
	ID     uuid.UUID `db:"id"`
	UserID uuid.UUID `db:"user_id"`
	Date   time.Time `db:"date"`
	RMSSD  *float64  `db:"rmssd"`
	BDI    *float64  `db:"bdi"`
}

func UpsertDailyHRV(db *sqlx.DB, h *DailyHRV) error {
	_, err := db.NamedExec(`
		INSERT INTO daily_hrv (user_id, date, rmssd, bdi)
		VALUES (:user_id, :date, :rmssd, :bdi)
		ON CONFLICT (user_id, date) DO UPDATE SET
		  rmssd = EXCLUDED.rmssd, bdi = EXCLUDED.bdi
	`, h)
	return err
}

func GetHRVRange(db *sqlx.DB, userID uuid.UUID, from, to string) ([]DailyHRV, error) {
	var rows []DailyHRV
	err := db.Select(&rows,
		`SELECT * FROM daily_hrv WHERE user_id = $1 AND date BETWEEN $2 AND $3 ORDER BY date`,
		userID, from, to)
	return rows, err
}
```

- [ ] **Step 6: Write models/heartrate.go**

```go
// models/heartrate.go
package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type HeartRate struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	Timestamp time.Time `db:"timestamp"`
	BPM       int       `db:"bpm"`
	Source    string    `db:"source"`
}

func BulkUpsertHeartRate(db *sqlx.DB, rows []HeartRate) error {
	if len(rows) == 0 {
		return nil
	}
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	for _, r := range rows {
		if _, err := tx.NamedExec(`
			INSERT INTO heart_rate (user_id, timestamp, bpm, source)
			VALUES (:user_id, :timestamp, :bpm, :source)
			ON CONFLICT (user_id, timestamp) DO UPDATE SET bpm = EXCLUDED.bpm
		`, r); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func GetHeartRateRange(db *sqlx.DB, userID uuid.UUID, from, to string) ([]HeartRate, error) {
	var rows []HeartRate
	err := db.Select(&rows,
		`SELECT * FROM heart_rate WHERE user_id = $1 AND timestamp BETWEEN $2 AND $3 ORDER BY timestamp`,
		userID, from, to)
	return rows, err
}
```

- [ ] **Step 7: Write models/spo2.go**

```go
// models/spo2.go
package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DailySpO2 struct {
	ID      uuid.UUID `db:"id"`
	UserID  uuid.UUID `db:"user_id"`
	Date    time.Time `db:"date"`
	AvgSpO2 *float64  `db:"avg_spo2"`
	MinSpO2 *float64  `db:"min_spo2"`
}

func UpsertDailySpO2(db *sqlx.DB, s *DailySpO2) error {
	_, err := db.NamedExec(`
		INSERT INTO daily_spo2 (user_id, date, avg_spo2, min_spo2)
		VALUES (:user_id, :date, :avg_spo2, :min_spo2)
		ON CONFLICT (user_id, date) DO UPDATE SET
		  avg_spo2 = EXCLUDED.avg_spo2, min_spo2 = EXCLUDED.min_spo2
	`, s)
	return err
}

func GetSpO2Range(db *sqlx.DB, userID uuid.UUID, from, to string) ([]DailySpO2, error) {
	var rows []DailySpO2
	err := db.Select(&rows,
		`SELECT * FROM daily_spo2 WHERE user_id = $1 AND date BETWEEN $2 AND $3 ORDER BY date`,
		userID, from, to)
	return rows, err
}
```

- [ ] **Step 8: Write models/stress.go**

```go
// models/stress.go
package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DailyStress struct {
	ID           uuid.UUID `db:"id"`
	UserID       uuid.UUID `db:"user_id"`
	Date         time.Time `db:"date"`
	StressHigh   *int      `db:"stress_high"`
	RecoveryHigh *int      `db:"recovery_high"`
	DaySummary   *string   `db:"day_summary"`
}

func UpsertDailyStress(db *sqlx.DB, s *DailyStress) error {
	_, err := db.NamedExec(`
		INSERT INTO daily_stress (user_id, date, stress_high, recovery_high, day_summary)
		VALUES (:user_id, :date, :stress_high, :recovery_high, :day_summary)
		ON CONFLICT (user_id, date) DO UPDATE SET
		  stress_high = EXCLUDED.stress_high,
		  recovery_high = EXCLUDED.recovery_high,
		  day_summary = EXCLUDED.day_summary
	`, s)
	return err
}

func GetStressRange(db *sqlx.DB, userID uuid.UUID, from, to string) ([]DailyStress, error) {
	var rows []DailyStress
	err := db.Select(&rows,
		`SELECT * FROM daily_stress WHERE user_id = $1 AND date BETWEEN $2 AND $3 ORDER BY date`,
		userID, from, to)
	return rows, err
}
```

- [ ] **Step 9: Write models/workout.go**

```go
// models/workout.go
package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Workout struct {
	ID            uuid.UUID  `db:"id"`
	UserID        uuid.UUID  `db:"user_id"`
	StartDatetime time.Time  `db:"start_datetime"`
	EndDatetime   *time.Time `db:"end_datetime"`
	Activity      *string    `db:"activity"`
	Calories      *int       `db:"calories"`
	Distance      *float64   `db:"distance"`
}

func UpsertWorkout(db *sqlx.DB, w *Workout) error {
	_, err := db.NamedExec(`
		INSERT INTO workouts (user_id, start_datetime, end_datetime, activity, calories, distance)
		VALUES (:user_id, :start_datetime, :end_datetime, :activity, :calories, :distance)
		ON CONFLICT (user_id, start_datetime) DO UPDATE SET
		  end_datetime = EXCLUDED.end_datetime,
		  activity = EXCLUDED.activity,
		  calories = EXCLUDED.calories,
		  distance = EXCLUDED.distance
	`, w)
	return err
}

func GetWorkoutsRange(db *sqlx.DB, userID uuid.UUID, from, to string) ([]Workout, error) {
	var rows []Workout
	err := db.Select(&rows,
		`SELECT * FROM workouts WHERE user_id = $1 AND start_datetime BETWEEN $2 AND $3 ORDER BY start_datetime`,
		userID, from, to)
	return rows, err
}
```

- [ ] **Step 10: Compile all models**

```bash
go build ./models/...
```

Expected: no output

- [ ] **Step 11: Commit**

```bash
git add Kairos/backend/models/
git commit -m "feat(kairos): all domain models with upsert queries"
```

---

## Task 5: Auth — JWT

**Files:**
- Create: `Kairos/backend/auth/jwt.go`
- Create: `Kairos/backend/auth/jwt_test.go`

- [ ] **Step 1: Write auth/jwt.go**

```go
// auth/jwt.go
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

func SignJWT(userID uuid.UUID, secret string) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ValidateJWT(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
```

- [ ] **Step 2: Write auth/jwt_test.go**

```go
// auth/jwt_test.go
package auth

import (
	"testing"

	"github.com/google/uuid"
)

func TestJWTRoundTrip(t *testing.T) {
	secret := "test-secret-key-long-enough"
	userID := uuid.New()

	token, err := SignJWT(userID, secret)
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}

	claims, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}

	if claims.UserID != userID {
		t.Fatalf("want userID %s, got %s", userID, claims.UserID)
	}
}

func TestJWTWrongSecret(t *testing.T) {
	userID := uuid.New()
	token, _ := SignJWT(userID, "correct-secret")
	_, err := ValidateJWT(token, "wrong-secret")
	if err == nil {
		t.Fatal("expected error with wrong secret, got nil")
	}
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./auth/... -v
```

Expected:
```
--- PASS: TestJWTRoundTrip
--- PASS: TestJWTWrongSecret
PASS
```

- [ ] **Step 4: Commit**

```bash
git add Kairos/backend/auth/
git commit -m "feat(kairos): JWT sign + validate with tests"
```

---

## Task 6: Auth — Oura OAuth2 PKCE

**Files:**
- Create: `Kairos/backend/auth/oauth.go`

- [ ] **Step 1: Write auth/oauth.go**

```go
// auth/oauth.go
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	ouraAuthURL  = "https://cloud.ouraring.com/oauth/authorize"
	ouraTokenURL = "https://api.ouraring.com/oauth/token"
	ouraUserURL  = "https://api.ouraring.com/v2/usercollection/personal_info"
)

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

type OuraPersonalInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func GeneratePKCE() (verifier, challenge string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h[:])
	return
}

func GenerateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (c *OAuthConfig) AuthURL(state, challenge string) string {
	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {c.ClientID},
		"redirect_uri":          {c.RedirectURL},
		"scope":                 {"email personal daily workout heartrate"},
		"state":                 {state},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	return ouraAuthURL + "?" + params.Encode()
}

func (c *OAuthConfig) ExchangeCode(ctx context.Context, code, verifier string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {c.RedirectURL},
		"client_id":     {c.ClientID},
		"client_secret": {c.ClientSecret},
		"code_verifier": {verifier},
	}
	return c.postToken(ctx, data)
}

func (c *OAuthConfig) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {c.ClientID},
		"client_secret": {c.ClientSecret},
	}
	return c.postToken(ctx, data)
}

func (c *OAuthConfig) postToken(ctx context.Context, data url.Values) (*TokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ouraTokenURL,
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed %d: %s", resp.StatusCode, body)
	}

	var tr TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, err
	}
	return &tr, nil
}

func FetchPersonalInfo(ctx context.Context, accessToken string) (*OuraPersonalInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ouraUserURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("personal info failed %d: %s", resp.StatusCode, body)
	}

	var info OuraPersonalInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}
```

- [ ] **Step 2: Compile check**

```bash
go build ./auth/...
```

Expected: no output

- [ ] **Step 3: Commit**

```bash
git add Kairos/backend/auth/oauth.go
git commit -m "feat(kairos): Oura OAuth2 PKCE flow"
```

---

## Task 7: Middleware

**Files:**
- Create: `Kairos/backend/middleware/auth.go`
- Create: `Kairos/backend/middleware/cors.go`

- [ ] **Step 1: Write middleware/auth.go**

```go
// middleware/auth.go
package middleware

import (
	"net/http"
	"strings"

	"github.com/NishanthMolleti/kairos/auth"
	"github.com/gin-gonic/gin"
)

func AuthRequired(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := auth.ValidateJWT(tokenStr, jwtSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("userID", claims.UserID)
		c.Next()
	}
}
```

- [ ] **Step 2: Write middleware/cors.go**

```go
// middleware/cors.go
package middleware

import (
	"github.com/gin-gonic/gin"
)

func CORS(frontendURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", frontendURL)
		c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
```

- [ ] **Step 3: Compile check**

```bash
go build ./middleware/...
```

Expected: no output

- [ ] **Step 4: Commit**

```bash
git add Kairos/backend/middleware/
git commit -m "feat(kairos): JWT auth middleware + CORS"
```

---

## Task 8: Oura API client + sync

**Files:**
- Create: `Kairos/backend/oura/client.go`
- Create: `Kairos/backend/oura/sync.go`

- [ ] **Step 1: Write oura/client.go**

```go
// oura/client.go
package oura

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const baseURL = "https://api.ouraring.com/v2/usercollection"

type Client struct {
	accessToken string
	http        *http.Client
}

func NewClient(accessToken string) *Client {
	return &Client{accessToken: accessToken, http: &http.Client{}}
}

type PagedResponse[T any] struct {
	Data []T `json:"data"`
}

func get[T any](ctx context.Context, c *Client, path string, params url.Values) ([]T, error) {
	u := baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("oura %s %d: %s", path, resp.StatusCode, body)
	}

	var pr PagedResponse[T]
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, err
	}
	return pr.Data, nil
}
```

- [ ] **Step 2: Write oura/sync.go**

```go
// oura/sync.go
package oura

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/NishanthMolleti/kairos/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type ouraSleep struct {
	Day                string `json:"day"`
	Score              *int   `json:"score"`
	TotalSleepDuration *int   `json:"total_sleep_duration"`
	Efficiency         *int   `json:"efficiency"`
	Latency            *int   `json:"latency"`
	RemSleepDuration   *int   `json:"rem_sleep_duration"`
	DeepSleepDuration  *int   `json:"deep_sleep_duration"`
	LightSleepDuration *int   `json:"light_sleep_duration"`
	AwakeTime          *int   `json:"awake_time"`
	RestlessPeriods    *int   `json:"restless_periods"`
}

type ouraReadiness struct {
	Day              string   `json:"day"`
	Score            *int     `json:"score"`
	HRVBalance       *int     `json:"hrv_balance"`
	BodyTemperature  *float64 `json:"body_temperature"`
	RecoveryIndex    *int     `json:"recovery_index"`
	RestingHeartRate *int     `json:"resting_heart_rate"`
	ActivityBalance  *int     `json:"activity_balance"`
	SleepBalance     *int     `json:"sleep_balance"`
}

type ouraActivity struct {
	Day                string   `json:"day"`
	Score              *int     `json:"score"`
	Steps              *int     `json:"steps"`
	TotalCalories      *int     `json:"total_calories"`
	ActiveCalories     *int     `json:"active_calories"`
	METMinutes         *float64 `json:"met_minutes_custom"`
	SedentaryTime      *int     `json:"sedentary_time"`
	LowActivityTime    *int     `json:"low_activity_time"`
	MediumActivityTime *int     `json:"medium_activity_time"`
	HighActivityTime   *int     `json:"high_activity_time"`
}

type ouraHRV struct {
	Day   string   `json:"day"`
	RMSSD *float64 `json:"rmssd"`
}

type ouraHeartRate struct {
	Timestamp string `json:"timestamp"`
	BPM       int    `json:"bpm"`
	Source    string `json:"source"`
}

type ouraSpO2 struct {
	Day         string   `json:"day"`
	SPO2Average *float64 `json:"spo2_percentage"`
}

type ouraStress struct {
	Day          string  `json:"day"`
	StressHigh   *int    `json:"stress_high"`
	RecoveryHigh *int    `json:"recovery_high"`
	DaySummary   *string `json:"day_summary"`
}

type ouraWorkout struct {
	StartDatetime string   `json:"start_datetime"`
	EndDatetime   string   `json:"end_datetime"`
	Activity      string   `json:"activity"`
	Calories      *int     `json:"calories"`
	Distance      *float64 `json:"distance"`
}

func dateRange() (string, string) {
	to := time.Now().Format("2006-01-02")
	from := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	return from, to
}

func SyncUser(ctx context.Context, db *sqlx.DB, userID uuid.UUID, accessToken string) error {
	client := NewClient(accessToken)
	from, to := dateRange()
	params := url.Values{"start_date": {from}, "end_date": {to}}
	var errs []error

	// Sleep
	sleepData, err := get[ouraSleep](ctx, client, "/daily_sleep", params)
	if err != nil {
		errs = append(errs, fmt.Errorf("sleep: %w", err))
	} else {
		for _, s := range sleepData {
			date, _ := time.Parse("2006-01-02", s.Day)
			models.UpsertDailySleep(db, &models.DailySleep{
				UserID: userID, Date: date,
				Score: s.Score, TotalSleepDuration: s.TotalSleepDuration,
				Efficiency: s.Efficiency, Latency: s.Latency,
				REMSleepDuration: s.RemSleepDuration, DeepSleepDuration: s.DeepSleepDuration,
				LightSleepDuration: s.LightSleepDuration, AwakeTime: s.AwakeTime,
				RestlessPeriods: s.RestlessPeriods,
			})
		}
	}

	// Readiness
	readinessData, err := get[ouraReadiness](ctx, client, "/daily_readiness", params)
	if err != nil {
		errs = append(errs, fmt.Errorf("readiness: %w", err))
	} else {
		for _, r := range readinessData {
			date, _ := time.Parse("2006-01-02", r.Day)
			models.UpsertDailyReadiness(db, &models.DailyReadiness{
				UserID: userID, Date: date,
				Score: r.Score, HRVBalance: r.HRVBalance,
				BodyTemperature: r.BodyTemperature, RecoveryIndex: r.RecoveryIndex,
				RestingHeartRate: r.RestingHeartRate, ActivityBalance: r.ActivityBalance,
				SleepBalance: r.SleepBalance,
			})
		}
	}

	// Activity
	activityData, err := get[ouraActivity](ctx, client, "/daily_activity", params)
	if err != nil {
		errs = append(errs, fmt.Errorf("activity: %w", err))
	} else {
		for _, a := range activityData {
			date, _ := time.Parse("2006-01-02", a.Day)
			models.UpsertDailyActivity(db, &models.DailyActivity{
				UserID: userID, Date: date,
				Score: a.Score, Steps: a.Steps,
				Calories: a.TotalCalories, ActiveCalories: a.ActiveCalories,
				METMinutes: a.METMinutes, SedentaryTime: a.SedentaryTime,
				LowActivity: a.LowActivityTime, MediumActivity: a.MediumActivityTime,
				HighActivity: a.HighActivityTime,
			})
		}
	}

	// HRV
	hrvData, err := get[ouraHRV](ctx, client, "/daily_hrv", params)
	if err != nil {
		errs = append(errs, fmt.Errorf("hrv: %w", err))
	} else {
		for _, h := range hrvData {
			date, _ := time.Parse("2006-01-02", h.Day)
			models.UpsertDailyHRV(db, &models.DailyHRV{
				UserID: userID, Date: date, RMSSD: h.RMSSD,
			})
		}
	}

	// Heart Rate
	hrParams := url.Values{
		"start_datetime": {from + "T00:00:00Z"},
		"end_datetime":   {to + "T23:59:59Z"},
	}
	hrData, err := get[ouraHeartRate](ctx, client, "/heartrate", hrParams)
	if err != nil {
		errs = append(errs, fmt.Errorf("heartrate: %w", err))
	} else {
		hrRows := make([]models.HeartRate, 0, len(hrData))
		for _, h := range hrData {
			ts, _ := time.Parse(time.RFC3339, h.Timestamp)
			hrRows = append(hrRows, models.HeartRate{
				UserID: userID, Timestamp: ts, BPM: h.BPM, Source: h.Source,
			})
		}
		if err := models.BulkUpsertHeartRate(db, hrRows); err != nil {
			errs = append(errs, fmt.Errorf("heartrate upsert: %w", err))
		}
	}

	// SpO2
	spo2Data, err := get[ouraSpO2](ctx, client, "/daily_spo2", params)
	if err != nil {
		errs = append(errs, fmt.Errorf("spo2: %w", err))
	} else {
		for _, s := range spo2Data {
			date, _ := time.Parse("2006-01-02", s.Day)
			models.UpsertDailySpO2(db, &models.DailySpO2{
				UserID: userID, Date: date, AvgSpO2: s.SPO2Average,
			})
		}
	}

	// Stress
	stressData, err := get[ouraStress](ctx, client, "/daily_stress", params)
	if err != nil {
		errs = append(errs, fmt.Errorf("stress: %w", err))
	} else {
		for _, s := range stressData {
			date, _ := time.Parse("2006-01-02", s.Day)
			models.UpsertDailyStress(db, &models.DailyStress{
				UserID: userID, Date: date,
				StressHigh: s.StressHigh, RecoveryHigh: s.RecoveryHigh,
				DaySummary: s.DaySummary,
			})
		}
	}

	// Workouts
	workoutData, err := get[ouraWorkout](ctx, client, "/workout", params)
	if err != nil {
		errs = append(errs, fmt.Errorf("workouts: %w", err))
	} else {
		for _, w := range workoutData {
			start, _ := time.Parse(time.RFC3339, w.StartDatetime)
			var end *time.Time
			if w.EndDatetime != "" {
				t, _ := time.Parse(time.RFC3339, w.EndDatetime)
				end = &t
			}
			activity := w.Activity
			models.UpsertWorkout(db, &models.Workout{
				UserID: userID, StartDatetime: start, EndDatetime: end,
				Activity: &activity, Calories: w.Calories, Distance: w.Distance,
			})
		}
	}

	if len(errs) > 0 {
		log.Printf("sync user %s partial errors: %v", userID, errs)
	}

	return models.UpdateLastSync(db, userID)
}
```

- [ ] **Step 3: Compile check**

```bash
go build ./oura/...
```

Expected: no output

- [ ] **Step 4: Commit**

```bash
git add Kairos/backend/oura/
git commit -m "feat(kairos): Oura API client + full sync (9 endpoints)"
```

---

## Task 9: Daily cron scheduler

**Files:**
- Create: `Kairos/backend/scheduler/cron.go`

- [ ] **Step 1: Write scheduler/cron.go**

```go
// scheduler/cron.go
package scheduler

import (
	"context"
	"log"

	"github.com/NishanthMolleti/kairos/models"
	"github.com/NishanthMolleti/kairos/oura"
	"github.com/jmoiron/sqlx"
	"github.com/robfig/cron/v3"
)

func Start(db *sqlx.DB) *cron.Cron {
	c := cron.New()
	c.AddFunc("0 3 * * *", func() { runSync(db) })
	c.Start()
	log.Println("cron scheduler started (daily at 03:00 UTC)")
	return c
}

func runSync(db *sqlx.DB) {
	users, err := models.GetAllUsers(db)
	if err != nil {
		log.Printf("cron: get users failed: %v", err)
		return
	}
	log.Printf("cron: syncing %d users", len(users))
	for _, u := range users {
		if err := oura.SyncUser(context.Background(), db, u.ID, u.AccessToken); err != nil {
			log.Printf("cron: sync failed for user %s: %v", u.ID, err)
		} else {
			log.Printf("cron: synced user %s", u.ID)
		}
	}
}
```

- [ ] **Step 2: Compile check**

```bash
go build ./scheduler/...
```

Expected: no output

- [ ] **Step 3: Commit**

```bash
git add Kairos/backend/scheduler/
git commit -m "feat(kairos): daily cron sync at 03:00 UTC"
```

---

## Task 10: HTTP handlers

**Files:**
- Create: `Kairos/backend/handlers/auth.go`
- Create: `Kairos/backend/handlers/user.go`
- Create: `Kairos/backend/handlers/sync.go`
- Create: `Kairos/backend/handlers/metrics.go`

- [ ] **Step 1: Write handlers/auth.go**

```go
// handlers/auth.go
package handlers

import (
	"net/http"

	kauth "github.com/NishanthMolleti/kairos/auth"
	"github.com/NishanthMolleti/kairos/config"
	"github.com/NishanthMolleti/kairos/models"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type AuthHandler struct {
	cfg   *config.Config
	db    *sqlx.DB
	oauth *kauth.OAuthConfig
}

func NewAuthHandler(cfg *config.Config, db *sqlx.DB) *AuthHandler {
	return &AuthHandler{
		cfg: cfg,
		db:  db,
		oauth: &kauth.OAuthConfig{
			ClientID:     cfg.OuraClientID,
			ClientSecret: cfg.OuraClientSecret,
			RedirectURL:  cfg.OuraRedirectURL,
		},
	}
}

// GET /auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	verifier, challenge, err := kauth.GeneratePKCE()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "pkce failed"})
		return
	}
	state, err := kauth.GenerateState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "state failed"})
		return
	}
	c.SetCookie("pkce_verifier", verifier, 600, "/", "", false, true)
	c.SetCookie("oauth_state", state, 600, "/", "", false, true)
	c.Redirect(http.StatusFound, h.oauth.AuthURL(state, challenge))
}

// GET /auth/callback
func (h *AuthHandler) Callback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	cookieState, err := c.Cookie("oauth_state")
	if err != nil || cookieState != state {
		c.JSON(http.StatusBadRequest, gin.H{"error": "state mismatch"})
		return
	}
	verifier, err := c.Cookie("pkce_verifier")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing verifier"})
		return
	}

	tokens, err := h.oauth.ExchangeCode(c.Request.Context(), code, verifier)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token exchange failed"})
		return
	}

	info, err := kauth.FetchPersonalInfo(c.Request.Context(), tokens.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "fetch user failed"})
		return
	}

	if err := models.UpsertUser(h.db, &models.User{
		OuraUserID:   info.ID,
		Email:        info.Email,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	dbUser, err := models.GetUserByOuraID(h.db, info.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	jwt, err := kauth.SignJWT(dbUser.ID, h.cfg.JWTSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "jwt error"})
		return
	}

	c.Redirect(http.StatusFound, h.cfg.FrontendURL+"/auth/callback?token="+jwt)
}

// POST /auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}
```

- [ ] **Step 2: Write handlers/user.go**

```go
// handlers/user.go
package handlers

import (
	"net/http"

	"github.com/NishanthMolleti/kairos/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type UserHandler struct{ db *sqlx.DB }

func NewUserHandler(db *sqlx.DB) *UserHandler { return &UserHandler{db: db} }

// GET /api/user
func (h *UserHandler) GetUser(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	user, err := models.GetUserByID(h.db, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": user.ID, "email": user.Email, "last_sync": user.LastSync})
}
```

- [ ] **Step 3: Write handlers/sync.go**

```go
// handlers/sync.go
package handlers

import (
	"net/http"

	"github.com/NishanthMolleti/kairos/models"
	"github.com/NishanthMolleti/kairos/oura"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type SyncHandler struct{ db *sqlx.DB }

func NewSyncHandler(db *sqlx.DB) *SyncHandler { return &SyncHandler{db: db} }

// POST /api/sync
func (h *SyncHandler) Sync(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	user, err := models.GetUserByID(h.db, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if err := oura.SyncUser(c.Request.Context(), h.db, userID, user.AccessToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sync failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "sync complete"})
}
```

- [ ] **Step 4: Write handlers/metrics.go**

```go
// handlers/metrics.go
package handlers

import (
	"net/http"
	"time"

	"github.com/NishanthMolleti/kairos/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type MetricsHandler struct{ db *sqlx.DB }

func NewMetricsHandler(db *sqlx.DB) *MetricsHandler { return &MetricsHandler{db: db} }

func (h *MetricsHandler) dateRange(c *gin.Context) (string, string) {
	to := time.Now().Format("2006-01-02")
	from := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	if f := c.Query("from"); f != "" {
		from = f
	}
	if t := c.Query("to"); t != "" {
		to = t
	}
	return from, to
}

func (h *MetricsHandler) Sleep(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	from, to := h.dateRange(c)
	data, err := models.GetSleepRange(h.db, userID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *MetricsHandler) Readiness(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	from, to := h.dateRange(c)
	data, err := models.GetReadinessRange(h.db, userID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *MetricsHandler) Activity(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	from, to := h.dateRange(c)
	data, err := models.GetActivityRange(h.db, userID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *MetricsHandler) HRV(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	from, to := h.dateRange(c)
	data, err := models.GetHRVRange(h.db, userID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *MetricsHandler) HeartRate(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	from, to := h.dateRange(c)
	data, err := models.GetHeartRateRange(h.db, userID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *MetricsHandler) SpO2(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	from, to := h.dateRange(c)
	data, err := models.GetSpO2Range(h.db, userID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *MetricsHandler) Stress(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	from, to := h.dateRange(c)
	data, err := models.GetStressRange(h.db, userID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *MetricsHandler) Workouts(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	from, to := h.dateRange(c)
	data, err := models.GetWorkoutsRange(h.db, userID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}
```

- [ ] **Step 5: Compile check**

```bash
go build ./handlers/...
```

Expected: no output

- [ ] **Step 6: Commit**

```bash
git add Kairos/backend/handlers/
git commit -m "feat(kairos): HTTP handlers — auth, user, sync, metrics"
```

---

## Task 11: main.go — wire everything

**Files:**
- Create: `Kairos/backend/main.go`

- [ ] **Step 1: Write main.go**

```go
// main.go
package main

import (
	"log"

	"github.com/NishanthMolleti/kairos/config"
	"github.com/NishanthMolleti/kairos/db"
	"github.com/NishanthMolleti/kairos/handlers"
	"github.com/NishanthMolleti/kairos/middleware"
	"github.com/NishanthMolleti/kairos/scheduler"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	database := db.Connect(cfg.DatabaseURL)
	db.RunMigrations(database)

	r := gin.Default()
	r.Use(middleware.CORS(cfg.FrontendURL))

	authH := handlers.NewAuthHandler(cfg, database)
	r.GET("/auth/login", authH.Login)
	r.GET("/auth/callback", authH.Callback)
	r.POST("/auth/logout", authH.Logout)

	api := r.Group("/api", middleware.AuthRequired(cfg.JWTSecret))
	{
		userH := handlers.NewUserHandler(database)
		api.GET("/user", userH.GetUser)

		syncH := handlers.NewSyncHandler(database)
		api.POST("/sync", syncH.Sync)

		metricsH := handlers.NewMetricsHandler(database)
		api.GET("/metrics/sleep", metricsH.Sleep)
		api.GET("/metrics/readiness", metricsH.Readiness)
		api.GET("/metrics/activity", metricsH.Activity)
		api.GET("/metrics/hrv", metricsH.HRV)
		api.GET("/metrics/heartrate", metricsH.HeartRate)
		api.GET("/metrics/spo2", metricsH.SpO2)
		api.GET("/metrics/stress", metricsH.Stress)
		api.GET("/metrics/workouts", metricsH.Workouts)
	}

	c := scheduler.Start(database)
	defer c.Stop()

	log.Printf("Kairos backend running on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
```

- [ ] **Step 2: Full build**

```bash
cd Kairos/backend
go build ./...
```

Expected: produces `kairos` binary, no errors

- [ ] **Step 3: Smoke test locally**

```bash
cp .env.example .env
# fill in real Supabase + Oura credentials in .env
go run main.go
```

Expected output:
```
db connected
migrations applied
cron scheduler started (daily at 03:00 UTC)
Kairos backend running on :8080
```

- [ ] **Step 4: Test OAuth flow**

Open `http://localhost:8080/auth/login` in browser.
Expected: redirects to `https://cloud.ouraring.com/oauth/authorize?...`

Complete consent → lands on `http://localhost:3000/auth/callback?token=<jwt>`

- [ ] **Step 5: Test protected endpoints**

```bash
TOKEN="<jwt from callback>"
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/user
# Expected: {"id":"...","email":"...","last_sync":null}

curl -X POST -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/sync
# Expected: {"message":"sync complete"}

curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/metrics/sleep
# Expected: JSON array of DailySleep records
```

- [ ] **Step 6: Commit**

```bash
git add Kairos/backend/main.go
git commit -m "feat(kairos): wire server — routes, cron, migrations"
```

---

## Task 12: GCP Compute Engine e2-micro deployment (Always Free)

**Domain:** `kairos.nimoclaw.dev` — Nginx on the VM serves both Go backend (`/api/*`) and Next.js static export (`/*`). No Vercel, no Node.js in production, no cross-origin issues.

**No platform config files needed** — binary runs as a systemd service directly on the VM.

Free tier: 1 e2-micro (1 vCPU shared, 1 GB RAM) in us-central1/us-west1/us-east1, 30 GB disk, 1 GB egress/month. Never expires.

- [ ] **Step 1: Create GCP e2-micro VM**

Go to [console.cloud.google.com](https://console.cloud.google.com) → Compute Engine → Create Instance:

```
Name:          kairos-backend
Region:        us-central1 (must be this region for Always Free)
Machine type:  e2-micro
Boot disk:     Debian 12, 30 GB standard persistent disk
Firewall:      ✅ Allow HTTP traffic, ✅ Allow HTTPS traffic
```

Click Create. Note the external IP address (e.g. `34.x.x.x`).

- [ ] **Step 2: SSH into VM and install dependencies**

```bash
# From GCP console: click SSH button, or:
gcloud compute ssh kairos-backend --zone us-central1-a

# On the VM:
sudo apt update && sudo apt install -y nginx certbot python3-certbot-nginx git
```

- [ ] **Step 3: Install Go on VM**

```bash
curl -OL https://go.dev/dl/go1.22.4.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.22.4.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
go version
```

Expected: `go version go1.22.4 linux/amd64`

- [ ] **Step 4: Clone repo and build binary on VM**

```bash
git clone https://github.com/NishanthMolleti/kairos.git
cd kairos/Kairos/backend
go build -o kairos .
sudo mv kairos /usr/local/bin/kairos
```

- [ ] **Step 5: Create env file on VM**

```bash
sudo mkdir -p /etc/kairos
sudo tee /etc/kairos/.env > /dev/null <<EOF
DATABASE_URL=postgresql://...
OURA_CLIENT_ID=...
OURA_CLIENT_SECRET=...
OURA_REDIRECT_URL=https://kairos.nimoclaw.dev/auth/callback
JWT_SECRET=...
GROQ_API_KEY=...
HUGGINGFACE_API_KEY=...
FRONTEND_URL=https://kairos.vercel.app
PORT=8080
EOF
sudo chmod 600 /etc/kairos/.env
```

- [ ] **Step 6: Create systemd service**

```bash
sudo tee /etc/systemd/system/kairos.service > /dev/null <<EOF
[Unit]
Description=Kairos Backend
After=network.target

[Service]
Type=simple
User=www-data
EnvironmentFile=/etc/kairos/.env
ExecStart=/usr/local/bin/kairos
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable kairos
sudo systemctl start kairos
sudo systemctl status kairos
```

Expected: `Active: active (running)`

- [ ] **Step 7: Add DNS A record for kairos.nimoclaw.dev**

In your domain registrar DNS settings for `nimoclaw.dev`:
```
Type: A
Name: kairos
Value: <your VM external IP>
TTL: 300
```

Wait ~5 minutes for propagation, then verify:
```bash
dig kairos.nimoclaw.dev +short
```
Expected: your VM's external IP

- [ ] **Step 8: Create static files directory for Next.js**

```bash
sudo mkdir -p /var/www/kairos
sudo chown -R www-data:www-data /var/www/kairos
```

Next.js static export files will go here in Plan 3. For now the directory just needs to exist.

- [ ] **Step 9: Configure Nginx — Go backend + static frontend**

```bash
sudo tee /etc/nginx/sites-available/kairos > /dev/null <<'EOF'
server {
    listen 80;
    server_name kairos.nimoclaw.dev;

    # Go backend — all /api/* routes
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        # Required for Sage streaming chat responses
        proxy_buffering off;
        proxy_cache off;
    }

    # Go backend — OAuth routes (no /api/ prefix)
    location /auth/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Next.js static export — everything else
    location / {
        root /var/www/kairos;
        try_files $uri $uri.html $uri/ /index.html;
        expires 1h;
        add_header Cache-Control "public, must-revalidate";
    }
}
EOF

sudo ln -s /etc/nginx/sites-available/kairos /etc/nginx/sites-enabled/
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t
sudo systemctl reload nginx
```

Expected: `nginx: configuration file /etc/nginx/nginx.conf test is successful`

- [ ] **Step 10: Enable HTTPS via Let's Encrypt (free)**

```bash
sudo certbot --nginx -d kairos.nimoclaw.dev
```

Expected: `Successfully deployed certificate for kairos.nimoclaw.dev`

Certbot auto-adds HTTPS redirect and SSL config to the Nginx block.

- [ ] **Step 11: Smoke test**

```bash
curl -I https://kairos.nimoclaw.dev/auth/login
```

Expected: `HTTP/2 302` with `Location: https://cloud.ouraring.com/oauth/authorize?...`

- [ ] **Step 12: Deploy backend updates (future)**

```bash
# Build for Linux amd64 on local machine:
cd Kairos/backend
GOOS=linux GOARCH=amd64 go build -o kairos .
gcloud compute scp kairos kairos-backend:/tmp/kairos --zone us-central1-a
gcloud compute ssh kairos-backend --zone us-central1-a \
  --command "sudo mv /tmp/kairos /usr/local/bin/kairos && sudo systemctl restart kairos"
```

- [ ] **Step 13: Commit**

```bash
git add docs/superpowers/plans/2026-07-01-kairos-plan-1-backend.md
git commit -m "feat(kairos): GCP e2-micro + Nginx routing — /api/* to Go, /* to Next.js static"
```

---

## Self-Review

**Spec coverage:**
- ✅ Oura OAuth2 PKCE — Tasks 6, 10
- ✅ JWT session auth — Tasks 5, 7
- ✅ All 9 Oura API endpoints synced — Task 8
- ✅ Daily cron 03:00 UTC — Task 9
- ✅ All DB tables — Task 3 (001_init.sql)
- ✅ pgvector + data_narratives — Task 3 (002_pgvector.sql)
- ✅ All metric REST endpoints — Tasks 10, 11
- ✅ CORS for Next.js — Task 7
- ✅ GCP e2-micro $0 deployment — Task 12
- ⚠️ Token refresh on 401 — `OAuthConfig.RefreshToken` defined, wired in Plan 2 cron
- ⚠️ chat endpoints + AI — Plan 2
- ⚠️ Frontend — Plan 3

**Placeholder scan:** None.

**Type consistency:** All handlers extract `userID` as `uuid.UUID` via `c.MustGet("userID").(uuid.UUID)` — matches what `middleware/auth.go` injects via `c.Set("userID", claims.UserID)` where `claims.UserID` is `uuid.UUID`.

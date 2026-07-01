# Kairos — Design Spec
**Date:** 2026-07-01  
**AI companion name:** Sage  
**Stack:** Go · Next.js · Postgres + pgvector · Groq (Llama 3.3 70B) · Oura OAuth2

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│                    KAIROS                           │
│                                                     │
│   Next.js Frontend (rich UI)                       │
│     Dashboard · Metrics · Chat (Sage)              │
└─────────────────────┬───────────────────────────────┘
                      │ HTTPS
┌─────────────────────▼───────────────────────────────┐
│   Go Backend                                        │
│                                                     │
│   ┌──────────┐  ┌──────────┐  ┌──────────────────┐ │
│   │  Auth    │  │  Sync    │  │  AI Query        │ │
│   │  OAuth2  │  │  Cron    │  │  Handler         │ │
│   │  Oura    │  │  daily   │  │  SQL gen → exec  │ │
│   └──────────┘  └──────────┘  │  RAG retrieval   │ │
│                                │  Groq stream     │ │
│                                └──────────────────┘ │
└──────────────┬──────────────────────────┬────────────┘
               │                          │
   ┌───────────▼─────────────┐  ┌─────────▼──────────┐
   │  Postgres + pgvector    │  │  Groq API          │
   │  - typed metric tables  │  │  Llama 3.3 70B     │
   │  - data_narratives      │  └────────────────────┘
   │    (embeddings)         │
   └─────────────────────────┘
               │
   ┌───────────▼─────────────┐
   │  Oura API v2            │
   │  (daily sync only)      │
   └─────────────────────────┘
```

---

## Core Principles

- **Telemetry-grounded answers only** — Sage answers exclusively from Oura data in the DB. No external knowledge, no inference beyond what the data contains. If data doesn't support an answer, Sage says so.
- **Multi-user** — each user authenticates via Oura OAuth2, owns their own data partition.
- **Daily sync** — background cron pulls all Oura endpoints once per day per user. 1-day latency acceptable.
- **Two-layer AI retrieval** — SQL for exact facts + RAG over narrative embeddings for trend/anomaly context.

---

## Backend (Go)

### Auth Layer
- Oura OAuth2 flow (authorization code + PKCE)
- Tokens stored per user in DB (access + refresh)
- Token refresh handled transparently on sync

### Sync Layer (Cron — daily per user)
Pulls all Oura API v2 endpoints:
- `/usercollection/daily_sleep`
- `/usercollection/daily_readiness`
- `/usercollection/daily_activity`
- `/usercollection/heartrate`
- `/usercollection/daily_hrv`
- `/usercollection/daily_spo2`
- `/usercollection/sleep` (detailed stages)
- `/usercollection/workout`
- `/usercollection/daily_stress`

After sync, generates narrative summaries per day, embeds via Groq embedding API, stores in `data_narratives` with pgvector.

### AI Query Layer
1. User sends question via chat
2. Sage system prompt enforces telemetry-only constraint
3. Groq generates SQL against user's data partition
4. SQL executes against Postgres — returns exact numbers
5. pgvector similarity search retrieves relevant narrative embeddings
6. Groq composes final answer from SQL results + narratives
7. Response streams back to frontend

---

## Database Schema

```sql
-- Users
users (
  id            UUID PRIMARY KEY,
  oura_user_id  TEXT UNIQUE,
  email         TEXT,
  access_token  TEXT,
  refresh_token TEXT,
  last_sync     TIMESTAMPTZ
)

-- Sleep
daily_sleep (
  id                    UUID PRIMARY KEY,
  user_id               UUID REFERENCES users,
  date                  DATE,
  score                 INT,
  total_sleep_duration  INT,  -- seconds
  efficiency            INT,  -- %
  latency               INT,  -- seconds
  rem_sleep_duration    INT,
  deep_sleep_duration   INT,
  light_sleep_duration  INT,
  awake_time            INT,
  restless_periods      INT
)

-- Readiness
daily_readiness (
  id                  UUID PRIMARY KEY,
  user_id             UUID REFERENCES users,
  date                DATE,
  score               INT,
  hrv_balance         INT,
  body_temperature    FLOAT,
  recovery_index      INT,
  resting_heart_rate  INT,
  activity_balance    INT,
  sleep_balance       INT
)

-- Activity
daily_activity (
  id               UUID PRIMARY KEY,
  user_id          UUID REFERENCES users,
  date             DATE,
  score            INT,
  steps            INT,
  calories         INT,
  active_calories  INT,
  met_minutes      FLOAT,
  sedentary_time   INT,
  low_activity     INT,
  medium_activity  INT,
  high_activity    INT
)

-- HRV
daily_hrv (
  id       UUID PRIMARY KEY,
  user_id  UUID REFERENCES users,
  date     DATE,
  rmssd    FLOAT,
  bdi      FLOAT   -- breathing disturbance index
)

-- Heart Rate (time-series)
heart_rate (
  id        UUID PRIMARY KEY,
  user_id   UUID REFERENCES users,
  timestamp TIMESTAMPTZ,
  bpm       INT,
  source    TEXT
)

-- SpO2
daily_spo2 (
  id          UUID PRIMARY KEY,
  user_id     UUID REFERENCES users,
  date        DATE,
  avg_spo2    FLOAT,
  min_spo2    FLOAT
)

-- Stress
daily_stress (
  id             UUID PRIMARY KEY,
  user_id        UUID REFERENCES users,
  date           DATE,
  stress_high    INT,  -- minutes
  recovery_high  INT,
  day_summary    TEXT
)

-- Workouts
workouts (
  id              UUID PRIMARY KEY,
  user_id         UUID REFERENCES users,
  start_datetime  TIMESTAMPTZ,
  end_datetime    TIMESTAMPTZ,
  activity        TEXT,
  calories        INT,
  distance        FLOAT
)

-- RAG Narratives (pgvector)
data_narratives (
  id          UUID PRIMARY KEY,
  user_id     UUID REFERENCES users,
  date        DATE,
  period_type TEXT,       -- 'daily' | 'weekly'
  content     TEXT,       -- e.g. "June 30: HRV 34ms (3rd lowest this month)..."
  embedding   VECTOR(1536)
)

-- Chat
chat_sessions (
  id         UUID PRIMARY KEY,
  user_id    UUID REFERENCES users,
  created_at TIMESTAMPTZ
)

chat_messages (
  id          UUID PRIMARY KEY,
  session_id  UUID REFERENCES chat_sessions,
  role        TEXT,  -- 'user' | 'assistant'
  content     TEXT,
  sql_used    TEXT,  -- stored for transparency/debugging
  created_at  TIMESTAMPTZ
)
```

---

## AI — Sage

**Model:** Llama 3.3 70B via Groq  
**Grounding constraint:** System prompt explicitly instructs Sage to answer only from provided SQL results and narrative context. If data is absent, respond: *"I don't have data to answer that."*

**Query pipeline:**
```
user question
     │
     ▼
[SQL generation] ──► postgres ──► exact rows
     │
     ▼
[RAG retrieval] ──► pgvector ──► relevant narratives
     │
     ▼
[Groq compose + stream] ──► grounded natural language answer
```

**Embeddings:** Groq embedding API (`nomic-embed-text` fallback for cost control).

---

## Frontend (Next.js)

Goal: richer than Oura's native app.

### Pages
- `/` — Dashboard: metric cards (sleep score, readiness, HRV, steps), trend sparklines, today's summary
- `/sleep` — Deep sleep analysis: stages chart, efficiency trend, latency history
- `/readiness` — Readiness breakdown: contributor rings, HRV trend, body temp delta
- `/activity` — Activity breakdown: step trend, calorie burn, activity intensity distribution
- `/heart` — Heart rate timeline, resting HR trend, HRV detail
- `/chat` — Sage chat interface: full conversation history, streamed responses, source transparency (shows SQL used)
- `/settings` — Account, sync status, disconnect Oura

### UI Principles
- Dark-first design with glass morphism accents
- Animated metric transitions (Framer Motion)
- Data viz: Recharts / Tremor
- Streaming chat with typing indicator
- "Source" toggle on each Sage answer — shows the SQL that produced it

---

## Deployment — $0 Stack

**Domain:** `kairos.nimoclaw.dev` — single domain, everything on one GCP VM.

Nginx routes:
- `/api/*` → Go backend (port 8080)
- `/*` → Next.js static export files (pre-built HTML/CSS/JS)

No Vercel. No Node.js running in production. No cross-origin issues.

| Service | Provider | Free Limit |
|---|---|---|
| Go backend + cron + static frontend | GCP Compute Engine e2-micro (Always Free) | 1 vCPU shared, 1GB RAM, 1GB egress/mo |
| Postgres + pgvector | Supabase free tier | 500MB DB, pgvector included |
| LLM (Sage) | Groq free tier | 1,000 req/day, 6K TPM |
| Embeddings | HuggingFace Inference API | nomic-embed-text, free |
| Oura API | Oura OAuth2 | Personal use, free |
| SSL cert | Let's Encrypt (Certbot) | Free, auto-renews |

**Total cost: $0/month.**

Note: Supabase free projects pause after 1 week inactivity. Daily sync cron prevents this naturally.

Next.js runs as static export (`output: 'export'` in next.config.ts). All data fetches are client-side to `/api/*`. No SSR needed.

Env vars: `OURA_CLIENT_ID`, `OURA_CLIENT_SECRET`, `GROQ_API_KEY`, `HUGGINGFACE_API_KEY`, `DATABASE_URL`, `JWT_SECRET`

---

## Out of Scope (v1)

- Push notifications / alerts
- Exporting data
- Comparing users
- Mobile app
- Voice interface

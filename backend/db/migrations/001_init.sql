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

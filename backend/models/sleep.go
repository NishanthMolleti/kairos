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

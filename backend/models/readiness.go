package models

import (
	"time"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DailyReadiness struct {
	ID               uuid.UUID `db:"id" json:"id"`
	UserID           uuid.UUID `db:"user_id" json:"user_id"`
	Date             time.Time `db:"date" json:"date"`
	Score            *int      `db:"score" json:"score"`
	HRVBalance       *int      `db:"hrv_balance" json:"hrv_balance"`
	BodyTemperature  *float64  `db:"body_temperature" json:"body_temperature"`
	RecoveryIndex    *int      `db:"recovery_index" json:"recovery_index"`
	RestingHeartRate *int      `db:"resting_heart_rate" json:"resting_heart_rate"`
	ActivityBalance  *int      `db:"activity_balance" json:"activity_balance"`
	SleepBalance     *int      `db:"sleep_balance" json:"sleep_balance"`
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
		  score = EXCLUDED.score, hrv_balance = EXCLUDED.hrv_balance,
		  body_temperature = EXCLUDED.body_temperature, recovery_index = EXCLUDED.recovery_index,
		  resting_heart_rate = EXCLUDED.resting_heart_rate,
		  activity_balance = EXCLUDED.activity_balance, sleep_balance = EXCLUDED.sleep_balance
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

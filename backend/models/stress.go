package models

import (
	"time"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DailyStress struct {
	ID           uuid.UUID `db:"id" json:"id"`
	UserID       uuid.UUID `db:"user_id" json:"user_id"`
	Date         time.Time `db:"date" json:"date"`
	StressHigh   *int      `db:"stress_high" json:"stress_high"`
	RecoveryHigh *int      `db:"recovery_high" json:"recovery_high"`
	DaySummary   *string   `db:"day_summary" json:"day_summary"`
}

func UpsertDailyStress(db *sqlx.DB, s *DailyStress) error {
	_, err := db.NamedExec(`
		INSERT INTO daily_stress (user_id, date, stress_high, recovery_high, day_summary)
		VALUES (:user_id, :date, :stress_high, :recovery_high, :day_summary)
		ON CONFLICT (user_id, date) DO UPDATE SET
		  stress_high = EXCLUDED.stress_high, recovery_high = EXCLUDED.recovery_high,
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

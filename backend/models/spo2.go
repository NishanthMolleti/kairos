package models

import (
	"time"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DailySpO2 struct {
	ID      uuid.UUID `db:"id" json:"id"`
	UserID  uuid.UUID `db:"user_id" json:"user_id"`
	Date    time.Time `db:"date" json:"date"`
	AvgSpO2 *float64  `db:"avg_spo2" json:"avg_spo2"`
	MinSpO2 *float64  `db:"min_spo2" json:"min_spo2"`
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
	rows := make([]DailySpO2, 0)
	err := db.Select(&rows,
		`SELECT * FROM daily_spo2 WHERE user_id = $1 AND date BETWEEN $2 AND $3 ORDER BY date`,
		userID, from, to)
	return rows, err
}

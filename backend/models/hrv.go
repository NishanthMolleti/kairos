package models

import (
	"time"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DailyHRV struct {
	ID     uuid.UUID `db:"id" json:"id"`
	UserID uuid.UUID `db:"user_id" json:"user_id"`
	Date   time.Time `db:"date" json:"date"`
	RMSSD  *float64  `db:"rmssd" json:"rmssd"`
	BDI    *float64  `db:"bdi" json:"bdi"`
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

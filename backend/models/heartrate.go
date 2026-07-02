package models

import (
	"time"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type HeartRate struct {
	ID        uuid.UUID `db:"id" json:"id"`
	UserID    uuid.UUID `db:"user_id" json:"user_id"`
	Timestamp time.Time `db:"timestamp" json:"timestamp"`
	BPM       int       `db:"bpm" json:"bpm"`
	Source    string    `db:"source" json:"source"`
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
	rows := make([]HeartRate, 0)
	err := db.Select(&rows,
		`SELECT * FROM heart_rate WHERE user_id = $1 AND timestamp BETWEEN $2 AND $3 ORDER BY timestamp`,
		userID, from, to)
	return rows, err
}

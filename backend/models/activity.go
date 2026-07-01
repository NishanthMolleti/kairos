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
		  score = EXCLUDED.score, steps = EXCLUDED.steps, calories = EXCLUDED.calories,
		  active_calories = EXCLUDED.active_calories, met_minutes = EXCLUDED.met_minutes,
		  sedentary_time = EXCLUDED.sedentary_time, low_activity = EXCLUDED.low_activity,
		  medium_activity = EXCLUDED.medium_activity, high_activity = EXCLUDED.high_activity
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

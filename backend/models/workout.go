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
		  end_datetime = EXCLUDED.end_datetime, activity = EXCLUDED.activity,
		  calories = EXCLUDED.calories, distance = EXCLUDED.distance
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

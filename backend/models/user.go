package models

import (
	"time"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type User struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	OuraUserID   string     `db:"oura_user_id" json:"oura_user_id"`
	Email        string     `db:"email" json:"email"`
	AccessToken  string     `db:"access_token" json:"-"`
	RefreshToken string     `db:"refresh_token" json:"-"`
	LastSync     *time.Time `db:"last_sync" json:"last_sync"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
}

func UpsertUser(db *sqlx.DB, u *User) error {
	_, err := db.NamedExec(`
		INSERT INTO users (oura_user_id, email, access_token, refresh_token)
		VALUES (:oura_user_id, :email, :access_token, :refresh_token)
		ON CONFLICT (oura_user_id) DO UPDATE
		  SET access_token  = EXCLUDED.access_token,
		      refresh_token = EXCLUDED.refresh_token,
		      email         = EXCLUDED.email
	`, u)
	return err
}

func GetUserByOuraID(db *sqlx.DB, ouraID string) (*User, error) {
	var u User
	err := db.Get(&u, `SELECT * FROM users WHERE oura_user_id = $1`, ouraID)
	return &u, err
}

func GetUserByID(db *sqlx.DB, id uuid.UUID) (*User, error) {
	var u User
	err := db.Get(&u, `SELECT * FROM users WHERE id = $1`, id)
	return &u, err
}

func GetAllUsers(db *sqlx.DB) ([]User, error) {
	var users []User
	err := db.Select(&users, `SELECT * FROM users`)
	return users, err
}

func UpdateLastSync(db *sqlx.DB, id uuid.UUID) error {
	_, err := db.Exec(`UPDATE users SET last_sync = NOW() WHERE id = $1`, id)
	return err
}

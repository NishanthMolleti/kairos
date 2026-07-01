package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type ChatSession struct {
	ID        uuid.UUID `db:"id" json:"id"`
	UserID    uuid.UUID `db:"user_id" json:"user_id"`
	Title     string    `db:"title" json:"title"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type ChatMessage struct {
	ID        uuid.UUID `db:"id" json:"id"`
	SessionID uuid.UUID `db:"session_id" json:"session_id"`
	Role      string    `db:"role" json:"role"`
	Content   string    `db:"content" json:"content"`
	SQLUsed   *string   `db:"sql_used" json:"sql_used,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func CreateSession(db *sqlx.DB, userID uuid.UUID, title string) (*ChatSession, error) {
	s := &ChatSession{}
	err := db.QueryRowx(
		`INSERT INTO chat_sessions (user_id, title, created_at)
		 VALUES ($1, $2, NOW())
		 RETURNING id, user_id, title, created_at`,
		userID, title,
	).StructScan(s)
	return s, err
}

func AddMessage(db *sqlx.DB, sessionID uuid.UUID, role, content string) (*ChatMessage, error) {
	m := &ChatMessage{}
	err := db.QueryRowx(
		`INSERT INTO chat_messages (session_id, role, content, created_at)
		 VALUES ($1, $2, $3, NOW())
		 RETURNING id, session_id, role, content, sql_used, created_at`,
		sessionID, role, content,
	).StructScan(m)
	return m, err
}

func GetSessionMessages(db *sqlx.DB, sessionID uuid.UUID) ([]ChatMessage, error) {
	var msgs []ChatMessage
	err := db.Select(&msgs,
		`SELECT id, session_id, role, content, sql_used, created_at
		 FROM chat_messages WHERE session_id = $1
		 ORDER BY created_at ASC`,
		sessionID,
	)
	return msgs, err
}

func GetSessionOwner(db *sqlx.DB, sessionID uuid.UUID) (uuid.UUID, error) {
	var ownerID uuid.UUID
	err := db.QueryRow(
		`SELECT user_id FROM chat_sessions WHERE id = $1`, sessionID,
	).Scan(&ownerID)
	return ownerID, err
}

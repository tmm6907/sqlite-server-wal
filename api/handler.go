package api

import (
	"os"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
)

type Handler struct {
	DB        *sqlx.DB
	Store     *sessions.CookieStore
	SessionID string
}

func NewHandler(db *sqlx.DB) *Handler {
	secretKey := uuid.New()
	return &Handler{
		DB:        db,
		Store:     sessions.NewCookieStore([]byte(secretKey.String())),
		SessionID: os.Getenv("SESSION_IDENTIFIER"),
	}
}

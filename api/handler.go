package api

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
)

type Handler struct {
	DB    *sqlx.DB
	Store *sessions.CookieStore
}

func NewHandler(db *sqlx.DB) *Handler {
	secretKey := uuid.New()
	return &Handler{
		DB:    db,
		Store: sessions.NewCookieStore([]byte(secretKey.String())),
	}
}

func (h *Handler) GetSessionKey(r *http.Request) (*sessions.Session, error) {
	return h.Store.Get(r, "session-key")
}

func (h *Handler) AttachUserDBs(userID int, userDB *sqlx.DB) error {
	rows, err := h.DB.Query("SELECT DISTINCT db_name, db_path FROM user_dbs WHERE user_id = ?", userID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var name string
		var file string
		if err := rows.Scan(&name, &file); err != nil {
			return err
		}
		query := fmt.Sprintf("ATTACH '%s' AS %s;", file, name)
		_, err = userDB.Exec(query)
		if err != nil {
			return err
		}
	}
	return nil
}

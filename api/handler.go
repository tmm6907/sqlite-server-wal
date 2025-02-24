package api

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	DB      *sqlx.DB
	Store   *sessions.CookieStore
	pkRegex *regexp.Regexp
}

type IndexInfo struct {
	Name    string `db:"name"`
	Unique  int    `db:"unique"`
	Seq     int    `db:"seq"`
	Origin  string `db:"origin"`
	Partial int    `db:"partial"`
}

func NewHandler(db *sqlx.DB) *Handler {
	secretKey := uuid.New()
	return &Handler{
		DB:      db,
		Store:   sessions.NewCookieStore([]byte(secretKey.String())),
		pkRegex: regexp.MustCompile(`(?i)SELECT\s+.*?\s+FROM\s+(\w+)`),
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

func (h *Handler) FindPK(c echo.Context, userDB *sqlx.DB, query string) ([]string, error) {
	found := h.pkRegex.FindStringSubmatch(query)
	if len(found) <= 1 {
		return nil, fmt.Errorf("table not found: %v", found)
	}
	tableName := found[1]
	c.Logger().Debug("Table: ", tableName)

	// Step 1: Check for PRIMARY KEY columns using PRAGMA table_info
	var columnNames []string
	if err := userDB.Select(&columnNames, fmt.Sprintf(`
		SELECT name FROM pragma_table_info('%s') WHERE pk > 0
	`, tableName)); err != nil {
		return nil, err
	}

	if len(columnNames) > 0 {
		return columnNames, nil
	}

	// Step 2: If no PK columns were found, check for a primary key index (for composite PKs)
	var indexName string
	if err := userDB.Get(&indexName, fmt.Sprintf(`
		SELECT name FROM pragma_index_list('%s') WHERE origin='u' LIMIT 1
	`, tableName)); err != nil {
		return nil, err
	}

	if err := userDB.Select(&columnNames, fmt.Sprintf(`
		SELECT name FROM pragma_index_info('%s')
	`, indexName)); err != nil {
		return nil, err
	}

	return columnNames, nil
}

func (h *Handler) GetUsername(r *http.Request) (string, error) {
	session, _ := h.Store.Get(r, "session-key")
	username, ok := session.Values["username"]
	if !ok {
		return "", fmt.Errorf("user not logged in")
	}
	return username.(string), nil
}

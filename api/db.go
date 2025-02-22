package api

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/tmm6907/sqlite-server-wal/models"
)

type CreateDBRequest struct {
	Name    string `json:"name"`
	Cache   string `json:"cache"`
	Journal string `json:"journal"`
	Sync    string `json:"sync"`
	Lock    string `json:"lock"`
}
type QueryRequest struct {
	Query string `json:"query"`
}

func (h *Handler) GetTables(c echo.Context) error {
	var tables []string
	var dbPath string
	session, _ := h.Store.Get(c.Request(), "session-key")
	username, ok := session.Values["username"]
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]any{
			"error": "not logged in",
		})
	}
	err := h.DB.QueryRow("SELECT db_path FROM users WHERE username = ?", username).Scan(&dbPath)
	if err != nil {
		return c.JSON(
			http.StatusInternalServerError,
			map[string]any{
				"error": err,
			},
		)
	}

	userDB, err := sqlx.Open("sqlite", fmt.Sprintf("db/%s", dbPath))
	if err != nil {
		return c.JSON(
			http.StatusInternalServerError,
			map[string]any{
				"error": err,
			},
		)
	}
	defer userDB.Close()
	var query string
	if c.Param("name") != "" {
		query = fmt.Sprintf("SELECT %s.name FROM sqlite_master WHERE type='table';", c.Param("name"))
	} else {
		query = "SELECT name FROM sqlite_master WHERE type='table';"
	}
	rows, err := userDB.Query(query)
	if err != nil {
		return c.JSON(
			http.StatusInternalServerError,
			map[string]any{
				"error": err,
			},
		)
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return c.JSON(
				http.StatusInternalServerError,
				map[string]any{
					"error": err.Error(),
				},
			)
		}
		tables = append(tables, name)
	}
	c.Logger().Debug(tables)
	return c.JSON(
		http.StatusOK,
		map[string]any{
			"tables": tables,
		},
	)
}

func (h *Handler) GetDatabases(c echo.Context) error {
	var dbs []string
	var dbPath string
	var id int
	session, _ := h.Store.Get(c.Request(), "session-key")
	username := session.Values["username"]
	c.Logger().Debug("Username: ", username)
	if username == nil {
		return c.JSON(
			http.StatusUnauthorized,
			map[string]any{
				"error": "user not logged in",
			},
		)
	}
	h.DB.QueryRow("SELECT id, db_path FROM users WHERE username = ?", username).Scan(&id, &dbPath)
	c.Logger().Debugf("USER INFO - id: %v, path: %s", id, dbPath)
	userDB, err := sqlx.Open("sqlite", fmt.Sprintf("db/%s", dbPath))
	if err != nil {
		c.Logger().Error(err)
		return c.JSON(
			http.StatusInternalServerError,
			map[string]any{
				"error": err,
			},
		)
	}
	defer userDB.Close()
	c.Logger().Debug("ID:", id)
	rows, err := h.DB.Query("SELECT DISTINCT db_name, db_path FROM user_dbs WHERE user_id = ?", id)
	if err != nil {
		c.Logger().Error(err)
		return c.JSON(
			http.StatusInternalServerError,
			map[string]any{
				"error": err,
			},
		)
	}
	dbs = []string{"main"}
	for rows.Next() {
		c.Logger().Debug("test")
		var name string
		var file string
		if err := rows.Scan(&name, &file); err != nil {
			c.Logger().Error(err)
			return c.JSON(
				http.StatusInternalServerError,
				map[string]any{
					"error": err.Error(),
				},
			)
		}
		query := fmt.Sprintf("ATTACH '%s' AS %s;", file, name)
		c.Logger().Debug(query)
		_, err = userDB.Exec(query)
		if err != nil {
			c.Logger().Error(err)
			return c.JSON(
				http.StatusInternalServerError,
				map[string]any{
					"error": err.Error(),
				},
			)
		}
		dbs = append(dbs, name)
	}
	c.Logger().Debug("DBs:", dbs)
	return c.JSON(
		http.StatusOK,
		map[string]any{
			"databases": dbs,
		},
	)
}

func (h *Handler) CreateDB(c echo.Context) error {
	var dbForm *CreateDBRequest
	var user models.User
	var count int

	if err := c.Bind(&dbForm); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}
	if dbForm.Cache == "" || dbForm.Journal == "" || dbForm.Lock == "" || dbForm.Name == "" || dbForm.Sync == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request. all fields are required"})
	}
	if strings.HasSuffix(dbForm.Name, ".db") {
		dbForm.Name = strings.Replace(dbForm.Name, ".db", "", 1)
	}
	rows, err := h.DB.Query("SELECT COUNT(*) FROM users WHERE db_path = ?;", dbForm.Name)
	if err != nil {
		c.Logger().Error(err)
		return c.JSON(
			http.StatusInternalServerError,
			map[string]any{
				"error": err.Error(),
			},
		)
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Scan(&count); err != nil || count > 0 {
			c.Logger().Error(err)
			return c.JSON(http.StatusInternalServerError, map[string]any{
				"msg": "database already exists",
				"err": err,
			})
		}
	}

	session, _ := h.Store.Get(c.Request(), "session-key")
	username := session.Values["username"]

	err = h.DB.Get(&user, "SELECT * FROM users WHERE username = ?", username)
	if err != nil {
		c.Logger().Error(err)
		return c.JSON(
			http.StatusInternalServerError,
			map[string]any{
				"error": err.Error(),
			})
	}
	c.Logger().Debugf("Current User db path: %v", user.DBPath)
	userDB, err := sqlx.Open("sqlite", fmt.Sprintf("db/%s", user.DBPath)) // open db to allow new db to be attatched

	if err != nil {
		c.Logger().Error(err)
		return c.JSON(
			http.StatusInternalServerError,
			map[string]any{
				"error": err.Error(),
			})
	}
	_, err = userDB.Exec("PRAGMA cache_size = -2000;") // Increase cache to allow larger memory use
	if err != nil {
		c.Logger().Errorf("Failed to set cache size: %v", err)
	}
	hostname := fmt.Sprintf("db/users/%v", username)

	dbPath := fmt.Sprintf("%s/%s.db", hostname, dbForm.Name)
	err = os.WriteFile(dbPath, []byte{}, PermOwnerReadWrite)
	if err != nil {
		return c.JSON(
			http.StatusInternalServerError,
			map[string]string{
				"error": fmt.Sprintf("failed to insert new user. %s", err.Error()),
			},
		)
	}

	_, err = h.DB.Exec("INSERT INTO user_dbs (user_id, db_name, db_path) VALUES (?,?,?);", user.ID, dbForm.Name, dbPath)
	if err != nil {
		c.Logger().Error(err)
		return c.JSON(
			http.StatusInternalServerError,
			map[string]any{
				"error": err.Error(),
			})
	}
	c.Logger().Debug("path:", dbPath)
	query := fmt.Sprintf("ATTACH '%s' AS %s;", dbPath, dbForm.Name)
	c.Logger().Debug(query)
	_, err = userDB.Exec(query)
	if err != nil {
		c.Logger().Error(err)
		return c.JSON(
			http.StatusInternalServerError,
			map[string]any{
				"error": err.Error(),
			},
		)
	}

	var config []string
	if dbForm.Cache == "Shared" {
		config = append(config, "PRAGMA cache=shared;")
	}
	if dbForm.Journal == "WAL" {
		config = append(config, "PRAGMA journal_mode=WAL;")
	}
	if dbForm.Sync == "Full" {
		config = append(config, "PRAGMA synchronous=FULL;")
	}
	if dbForm.Sync == "Off" {
		config = append(config, "PRAGMA synchronous=OFF;")
	}
	if dbForm.Lock == "Exclusive" {
		config = append(config, "PRAGMA locking_mode=EXCLUSIVE;")
	}

	_, err = userDB.Exec(strings.Join(config, " "))
	if err != nil {
		return c.JSON(
			http.StatusInternalServerError,
			map[string]any{
				"error": err.Error(),
			},
		)
	}

	return c.JSON(
		http.StatusCreated,
		map[string]any{
			"message": fmt.Sprintf("%s has been completed successfully", dbForm.Name),
		},
	)
}

package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/tmm6907/sqlite-server-wal/models"
	"github.com/tmm6907/sqlite-server-wal/util"
)

type CreateDBRequest struct {
	Name    string `json:"name"`
	Cache   string `json:"cache"`
	Journal string `json:"journal"`
	Sync    string `json:"sync"`
	Lock    string `json:"lock"`
}

func (h *Handler) GetNavData(c echo.Context) error {
	navData := make(chan map[string][]string)
	errChan := make(chan error)
	go func() {
		var dbPath string
		var id int

		var dbNames []string
		data := make(map[string][]string)
		username, err := h.GetUsername(c.Request())
		if err != nil {
			c.Logger().Error(err)
			errChan <- err
		}
		if err = h.DB.QueryRow("SELECT id, db_path FROM users WHERE username = ?", username).Scan(&id, &dbPath); err != nil {
			c.Logger().Error(err)
			errChan <- err
		}
		err = h.DB.Select(&dbNames, "SELECT DISTINCT db_name FROM user_dbs WHERE user_id = ?", id)
		if err != nil {
			c.Logger().Error(err)
			errChan <- err
		}

		userDB, err := sqlx.Open("sqlite", fmt.Sprintf("db/%s", dbPath))
		if err != nil {
			c.Logger().Error(err)
			errChan <- err
		}
		defer userDB.Close()
		if err = h.AttachUserDBs(id, userDB); err != nil {
			c.Logger().Error(err)
			errChan <- err
		}
		var tables []string
		err = userDB.Select(&tables, "SELECT name FROM sqlite_master WHERE type='table';")
		if err != nil {
			c.Logger().Error(err)
			errChan <- err
		}
		data["main"] = tables

		for _, db := range dbNames {
			err = userDB.Select(&tables, fmt.Sprintf("SELECT name FROM %s.sqlite_master WHERE type='table';", db))
			if err != nil {
				c.Logger().Errorf("Error loading query: %v", err)
				errChan <- err
			}
			data[db] = tables
		}
		navData <- data
	}()

	select {
	case data := <-navData:
		c.Logger().Debug(data)
		return c.JSON(
			http.StatusAccepted,
			map[string]any{
				"results": data,
			},
		)
	case err := <-errChan:
		return c.JSON(
			http.StatusInternalServerError,
			map[string]any{
				"error": err,
			},
		)
	}
}

// func (h *Handler) GetTables(c echo.Context) error {
// 	var tables []string
// 	var dbPath string
// 	var id int
// 	session, err := h.Store.Get(c.Request(), "session-key")
// 	if err != nil {
// 		return c.JSON(http.StatusUnauthorized, map[string]any{
// 			"error": "not logged in",
// 		})
// 	}
// 	username, ok := session.Values["username"]
// 	if !ok {
// 		return c.JSON(http.StatusUnauthorized, map[string]any{
// 			"error": "not logged in",
// 		})
// 	}

// 	if err = h.DB.QueryRow("SELECT id, db_path FROM users WHERE username = ?", username).Scan(&id, &dbPath); err != nil {
// 		return c.JSON(
// 			http.StatusInternalServerError,
// 			map[string]any{
// 				"error": err,
// 			},
// 		)
// 	}

// 	userDB, err := sqlx.Open("sqlite", fmt.Sprintf("db/%s", dbPath))
// 	if err != nil {
// 		return c.JSON(
// 			http.StatusInternalServerError,
// 			map[string]any{
// 				"error": err,
// 			},
// 		)
// 	}
// 	defer userDB.Close()

// 	if err = h.AttachUserDBs(id, userDB); err != nil {
// 		c.Logger().Error(err)
// 		return c.JSON(
// 			http.StatusInternalServerError,
// 			map[string]any{
// 				"error": err,
// 			},
// 		)
// 	}
// 	var query string
// 	if c.QueryParam("name") != "" {
// 		name := c.QueryParam("name")
// 		c.Logger().Debugf("Name: %s", name)
// 		query = fmt.Sprintf("SELECT name FROM %s.sqlite_master WHERE type='table';", name)
// 	} else {
// 		query = "SELECT name FROM sqlite_master WHERE type='table';"
// 	}
// 	rows, err := userDB.Query(query)
// 	if err != nil {
// 		c.Logger().Errorf("Error loading query: %v", err)
// 		return c.JSON(
// 			http.StatusInternalServerError,
// 			map[string]any{
// 				"error": err,
// 			},
// 		)
// 	}
// 	for rows.Next() {
// 		var name string
// 		if err := rows.Scan(&name); err != nil {
// 			return c.JSON(
// 				http.StatusInternalServerError,
// 				map[string]any{
// 					"error": err.Error(),
// 				},
// 			)
// 		}
// 		tables = append(tables, name)
// 	}
// 	c.Logger().Debugf("Tables: %v", tables)
// 	return c.JSON(
// 		http.StatusOK,
// 		map[string]any{
// 			"tables": tables,
// 		},
// 	)
// }

// func (h *Handler) GetDatabases(c echo.Context) error {
// 	var dbs []string
// 	var dbPath string
// 	var id int

// 	rows, err := h.DB.Query("SELECT DISTINCT db_name, db_path FROM user_dbs WHERE user_id = ?", id)
// 	if err != nil {
// 		c.Logger().Error(err)
// 		return c.JSON(
// 			http.StatusInternalServerError,
// 			map[string]any{
// 				"error": err,
// 			},
// 		)
// 	}
// 	dbs = []string{"main"}
// 	for rows.Next() {
// 		var name string
// 		var file string
// 		if err := rows.Scan(&name, &file); err != nil {
// 			c.Logger().Error(err)
// 			return c.JSON(
// 				http.StatusInternalServerError,
// 				map[string]any{
// 					"error": err.Error(),
// 				},
// 			)
// 		}
// 		dbs = append(dbs, name)
// 	}
// 	c.Logger().Debug("DBs:", dbs)
// 	return c.JSON(
// 		http.StatusOK,
// 		map[string]any{
// 			"databases": dbs,
// 		},
// 	)
// }

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

func (h *Handler) ImportDB(c echo.Context) error {
	form, err := c.MultipartForm()
	if err != nil {
		c.Logger().Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Failed to parse form"})
	}
	files := form.File["files"]
	if len(files) == 0 {
		c.Logger().Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "No file imported"})
	}
	session, _ := h.Store.Get(c.Request(), "session-key")
	username, ok := session.Values["username"]
	if !ok {
		c.Logger().Error("not logged in")
		return c.JSON(
			http.StatusUnauthorized,
			map[string]any{
				"error": "not logged in",
			},
		)
	}
	var userID int
	if err := h.DB.Get(&userID, "SELECT id FROM users WHERE username = ?;", username); err != nil {
		c.Logger().Error(err)
		return c.JSON(
			http.StatusInternalServerError,
			map[string]any{
				"error": err,
			},
		)
	}
	for _, file := range files {
		fmt.Println("Received file:", file.Filename)
		if err = util.ImportDBFile(c, file, username.(string)); err != nil {
			c.Logger().Error(err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Could not import file"})
		}
		h.DB.Exec(
			"INSERT OR REPLACE INTO user_dbs (user_id, db_name, db_path) VALUES (?, ?, ?)",
			userID, strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename)),
			fmt.Sprintf("db/users/%s/%s.db",
				username,
				strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename)),
			),
		)
	}
	return c.JSON(http.StatusAccepted, map[string]string{"message": "file imported successfully"})
}

func (h *Handler) ExportDB(c echo.Context) error {
	var dbPath string
	var id int
	dbName := c.QueryParam("db")
	fileType := c.QueryParam("type")
	c.Logger().Debug("Exporting DB")
	if dbName == "" || fileType == "" {
		c.Logger().Error("missing db or type")
		c.JSON(http.StatusBadRequest, map[string]string{"error": "must provide name and type"})
	}
	session, _ := h.Store.Get(c.Request(), "session-key")
	username, ok := session.Values["username"]
	if !ok {
		c.Logger().Error("not logged in")
		return c.JSON(
			http.StatusUnauthorized,
			map[string]any{
				"error": "not logged in",
			},
		)
	}
	if err := h.DB.QueryRow("SELECT id, db_path FROM users WHERE username = ?", username).Scan(&id, &dbPath); err != nil {
		return c.JSON(
			http.StatusInternalServerError,
			map[string]any{
				"error": err,
			},
		)
	}
	dbPath = "db/" + dbPath
	if dbName != "main" {
		if err := h.DB.Get(&dbPath, "SELECT db_path from user_dbs WHERE user_id = ? ", id); err != nil {
			c.Logger().Error(err)
			return c.JSON(
				http.StatusInternalServerError,
				map[string]any{
					"error": "couldn't locate db",
				},
			)
		}
	}
	c.Logger().Debug(dbPath)
	file, err := util.ExportDBFile(c, dbPath, dbName, fileType)

	if err != nil {
		c.Logger().Error(err)
		return c.JSON(
			http.StatusInternalServerError,
			map[string]any{
				"error": "couldn't export db",
			},
		)
	}
	c.Logger().Debug("Exporting DB: ", file.Name())
	defer os.Remove(file.Name())
	return c.Attachment(file.Name(), filepath.Base(file.Name()))
}

package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/tmm6907/sqlite-server-wal/models"
	"github.com/tmm6907/sqlite-server-wal/util"
)

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

const (
	// Read-write for the owner, read-only for others
	PermOwnerReadWrite = 0o644
	// Read-write-execute for the owner, read-only for others
	// PermOwnerReadWriteExec = 0o755
)

func (h *Handler) Login(c echo.Context) error {
	var user models.User
	var creds Credentials
	if err := c.Bind(&creds); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}
	if creds.Username == "" || creds.Password == "" {
		return c.JSON(
			http.StatusBadRequest,
			map[string]string{
				"error": "username and password are required",
			},
		)
	}

	err := h.DB.Get(&user, "SELECT * from users WHERE username = ?", creds.Username)
	if err != nil {
		return c.JSON(
			http.StatusBadRequest,
			map[string]string{
				"error": "username not recognized",
			},
		)
	}

	if !user.ValidatePassword(creds.Password) {
		return c.JSON(
			http.StatusInternalServerError,
			map[string]string{
				"error": "Failed to save session",
			},
		)
	}

	session, _ := h.Store.Get(c.Request(), "session-key")
	session.Values["username"] = creds.Username
	err = session.Save(c.Request(), c.Response())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to save session. %s", err.Error()),
		})
	}
	return c.JSON(
		http.StatusOK,
		map[string]string{
			"message": "Logged in!",
		},
	)
}

func (h *Handler) SignUp(c echo.Context) error {
	var user models.User
	var creds Credentials
	if err := c.Bind(&creds); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}
	passwordHash, err := util.HashPassword(creds.Password)
	if err != nil {
		return c.JSON(
			http.StatusInternalServerError,
			map[string]string{
				"error": "System Error",
			},
		)
	}
	dbPath := fmt.Sprintf("db/users/%s/root.db", creds.Username)
	err = os.WriteFile(dbPath, []byte{}, PermOwnerReadWrite)
	if err != nil {
		return c.JSON(
			http.StatusInternalServerError,
			map[string]string{
				"error": fmt.Sprintf("failed to insert new user. %s", err.Error()),
			},
		)
	}
	err = h.DB.Get(&user, "SELECT * from users WHERE username = ?", creds.Username)
	if err != nil {
		if _, err := h.DB.Exec("INSERT INTO users (username, password_hash, db_path) VALUES (?, ?, ?)", creds.Username, passwordHash, dbPath); err != nil {
			return c.JSON(
				http.StatusInternalServerError,
				map[string]string{
					"error": fmt.Sprintf("failed to insert new user. %s", err.Error()),
				},
			)
		}
		c.Logger().Info("%s has signed up!", creds.Username)
		session, _ := h.Store.Get(c.Request(), "session-key")
		session.Values["username"] = creds.Username
		err = session.Save(c.Request(), c.Response())
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to save session. %s", err.Error()),
			})
		}
		return c.JSON(
			http.StatusOK,
			map[string]string{
				"message": "Logged in!",
			},
		)
	}

	return c.JSON(
		http.StatusBadRequest,
		map[string]string{
			"error": "username already associated with an existing account",
		},
	)
}

func (h *Handler) IsAuth(c echo.Context) error {
	session, _ := h.GetSessionKey(c.Request())
	username, ok := session.Values["username"].(string)
	if !ok || username == "" {
		c.Logger().Info("not logged in")
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "You are not logged in",
		})
	}
	c.Logger().Debug("logged in")
	return c.JSON(
		http.StatusOK,
		map[string]string{
			"message": "You are logged in!",
		},
	)
}

func (h *Handler) GetUsers(c echo.Context) error {
	body := map[string]any{
		"error": NotImplemented{}.Error(),
	}
	return c.JSON(
		http.StatusOK,
		body,
	)
}

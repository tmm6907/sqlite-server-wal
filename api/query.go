package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/tmm6907/sqlite-server-wal/models"
	"github.com/tmm6907/sqlite-server-wal/util"
)

func (h *Handler) Query(c echo.Context) error {
	var query *QueryRequest
	var user models.User
	if err := c.Bind(&query); err != nil {
		c.Logger().Error(err)
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "Invalid request"})
	}
	if query.Query == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "Query must not be empty"})
	}
	if util.ContainsAttachStatement(query.Query) {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "ATTACH command not permitted"})
	}

	session, err := h.GetSessionKey(c.Request())

	if err != nil {
		c.Logger().Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "internal server error"})
	}

	username, ok := session.Values["username"]
	if !ok {
		return c.JSON(http.StatusNetworkAuthenticationRequired, map[string]any{"error": "not logged in"})
	}

	h.DB.Get(&user, "SELECT * FROM users WHERE username = ?", username)

	userDB, err := sqlx.Open("sqlite", fmt.Sprintf("db/%s", user.DBPath))
	if err != nil {
		c.Logger().Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "internal server error"})
	}
	err = h.AttachUserDBs(user.ID, userDB)
	if err != nil {
		c.Logger().Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "internal server error"})
	}
	if strings.HasPrefix(strings.ToUpper(query.Query), "SELECT") {
		rows, err := userDB.Query(query.Query)
		if err != nil {
			c.Logger().Error(err)
			return c.JSON(http.StatusBadRequest, map[string]any{"error": err})
		}
		columns, _ := rows.Columns()

		var res []map[string]interface{}
		for rows.Next() {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))

			// Assign each pointer to the corresponding interface
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			// Scan the row into the value pointers
			if err := rows.Scan(valuePtrs...); err != nil {
				c.Logger().Error(err)
				return c.JSON(http.StatusInternalServerError, map[string]any{"error": "internal server error"})
			}

			// Create a map for the current row
			rowMap := make(map[string]interface{})
			for i, col := range columns {
				// Dereference the value
				rowMap[col] = values[i]
			}

			res = append(res, rowMap)
		}
		c.Logger().Debugf("Results: %s", res)
		if res == nil {
			res = make([]map[string]interface{}, len(columns)) // Convert nil to empty slice
			colMap := make(map[string]any)
			for _, col := range columns {
				// Dereference the value
				colMap[col] = nil
			}
			res[0] = colMap
		}
		return c.JSON(http.StatusAccepted, map[string]any{"results": res})

	} else {
		result, err := userDB.Exec(query.Query)
		if err != nil {
			c.Logger().Error(err)
			return c.JSON(http.StatusBadRequest, map[string]any{"error": err})
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			c.Logger().Error(err)
			return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Could not retrieve rows affected"})
		}
		return c.JSON(
			http.StatusAccepted,
			map[string]any{
				"rowsAffected": rowsAffected,
			},
		)
	}
}

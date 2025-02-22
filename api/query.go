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

type QueryRequest struct {
	Query string `json:"query"`
	DB    string `json:"db"`
}

func (h *Handler) Query(c echo.Context) error {
	var query *QueryRequest
	var user models.User
	var targetDBPath string
	var connectionStr string
	if err := c.Bind(&query); err != nil {
		c.Logger().Error(err)
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "Invalid request"})
	}
	if query.Query == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "Query must not be empty"})
	}
	q, found := util.ContainsAttachStatement(query.Query)
	if found {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "ATTACH command not permitted"})
	}
	query.Query = q

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

	if query.DB != "" {
		if err = h.DB.Get(&targetDBPath, "SELECT db_path FROM user_dbs WHERE db_name = ?", query.DB); err != nil {
			c.Logger().Error(err)
			return c.JSON(http.StatusInternalServerError, map[string]any{"error": "internal server error"})
		}
		connectionStr = targetDBPath
	} else {
		targetDBPath = user.DBPath
		connectionStr = fmt.Sprintf("db/%s", targetDBPath)
	}
	c.Logger().Debugf("Connection string: %s", connectionStr)
	userDB, err := sqlx.Open("sqlite", connectionStr)
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
		c.Logger().Debugf("Query: %s", query.Query)
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
		var indexes []string
		indexes, err = h.FindPK(c, userDB, query.Query)
		if err != nil {
			c.Logger().Error(err)
			return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Could not retrieve primary keys"})
		}
		c.Logger().Debug(indexes)

		return c.JSON(http.StatusAccepted, map[string]any{
			"pks":     indexes,
			"results": res,
		})

	} else {
		c.Logger().Debugf("EXECUTION: %s", query.Query)
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
		c.Logger().Debugf("Rows affected: %d", rowsAffected)
		return c.JSON(
			http.StatusAccepted,
			map[string]any{
				"rowsAffected": rowsAffected,
			},
		)
	}
}

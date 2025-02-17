package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/tmm6907/sqlite-server-wal/api"
	"github.com/tmm6907/sqlite-server-wal/db"
)

func main() {
	db, err := db.Init()
	if err != nil {
		log.Fatalf("Failed to initialize db: %v", err)
	}
	sqliteDB := db.DB
	defer sqliteDB.Close()

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatalln("unable port not set")
	}

	s := echo.New()
	s.Debug = true
	s.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello World!")
	})
	routes := s.Group("/api")
	h := api.NewHandler(db)
	routes.GET("/users", h.GetUsers)
	routes.POST("/login", h.Login)
	routes.POST("/signup", h.SignUp)
	routes.GET("/auth", h.IsAuth)

	log.Fatalln(s.Start(fmt.Sprintf(":%s", port)))
}

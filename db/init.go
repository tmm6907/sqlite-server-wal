package db

import (
	"log"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

const DB_PATH = "/src/db/wal.db"

func SetWALMode(db *sqlx.DB) error {
	var mode string
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return err
	}
	log.Printf("Current journal mode: %s", mode)
	return nil
}

func Init() (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlite", DB_PATH)
	if err != nil {
		return nil, err
	}
	if err := SetWALMode(db); err != nil {

		return nil, err
	}
	log.Println("DB connection made successfully!")
	return db, nil
}

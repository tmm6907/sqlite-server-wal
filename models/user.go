package models

import (
	"time"

	"github.com/tmm6907/sqlite-server-wal/util"
)

type User struct {
	ID           int       `db:"id"`
	Username     string    `db:"username"`
	PasswordHash string    `db:"password_hash"`
	DBPath       string    `db:"db_path"`
	Created_at   time.Time `db:"created_at"`
}

func (u User) ValidatePassword(password string) bool {
	hashPass, err := util.HashPassword(password)
	if err != nil {
		return false
	}
	return u.PasswordHash == hashPass
}

package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID         int
	FirstName  string
	LastName   string
	Username   string
	Email      string
	Password   []byte
	DateJoined time.Time
	LastLogin  time.Time
}

type UserModel interface {
	Authenticate(username, password string) (int, error)
}

type userModel struct {
	DB *sql.DB
}

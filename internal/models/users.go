package models

import (
	"database/sql"
	"errors"
	"golang.org/x/crypto/bcrypt"
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

func NewUserModel(db *sql.DB) UserModel {
	return &userModel{DB: db}
}

func (um userModel) Authenticate(username, password string) (int, error) {
	var id int
	var hashedPassword []byte

	stmt := `SELECT id, password FROM users WHERE username = $1`

	err := um.DB.QueryRow(stmt, username).Scan(&id, &hashedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrInvalidCredentials
		}
		return 0, err
	}

	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return 0, ErrInvalidCredentials
		}

		return 0, err
	}

	stmt = `UPDATE users SET last_login = $1 WHERE id = $2`

	_, err = um.DB.Exec(stmt, time.Now(), id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

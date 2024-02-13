package models

import (
	"database/sql"
	"errors"
	"time"
)

type Swim struct {
	Id         int
	Date       time.Time
	DistanceM  int
	Assessment int
}

type SwimModel interface {
	Get(id int) (*Swim, error)
}

type swimModel struct {
	DB *sql.DB
}

func NewSwimModel(db *sql.DB) SwimModel {
	return &swimModel{DB: db}
}

func (sw *swimModel) Get(id int) (*Swim, error) {
	stmt := `SELECT id, date, distance_m, assessment FROM swims WHERE id = $1`

	row := sw.DB.QueryRow(stmt, id)

	var s Swim

	err := row.Scan(&s.Id, &s.Date, &s.DistanceM, &s.Assessment)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &Swim{}, ErrNoRecord
		} else {
			return &Swim{}, err
		}
	}

	return &s, nil
}

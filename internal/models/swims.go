package models

import (
	"database/sql"
)

type Swim struct {
	Id         int
	Date       string
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
	stmt := `SELECT id, date, distance_m, assessment FROM swims WHERE id = ?`

	row := sw.DB.QueryRow(stmt, id)

	var s Swim

	err := row.Scan(&s.Id, &s.Date, &s.DistanceM, &s.Assessment)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

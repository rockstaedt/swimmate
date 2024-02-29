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

type SwimSummary struct {
	TotalDistance   int
	TotalCount      int
	MonthlyDistance int
	MonthlyCount    int
	WeeklyDistance  int
	WeeklyCount     int
}

type SwimModel interface {
	Get() (*Swim, error)
	GetAll() ([]*Swim, error)
	Insert(date time.Time, distanceM int, assessment int) error
	Summarize() *SwimSummary
}

type swimModel struct {
	DB *sql.DB
}

func NewSwimModel(db *sql.DB) SwimModel {
	return &swimModel{DB: db}
}

func (sw *swimModel) Get() (*Swim, error) {
	stmt := `SELECT date, distance_m, assessment FROM tracks_track ORDER BY date ASC LIMIT 1;`

	row := sw.DB.QueryRow(stmt)

	var s Swim

	err := row.Scan(&s.Date, &s.DistanceM, &s.Assessment)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &Swim{}, ErrNoRecord
		} else {
			return &Swim{}, err
		}
	}

	return &s, nil
}

func (sw *swimModel) GetAll() ([]*Swim, error) {
	stmt := `SELECT date, distance_m, assessment FROM tracks_track WHERE user_id = $1 ORDER BY date ASC;`

	rows, err := sw.DB.Query(stmt, 1)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var swims []*Swim
	for rows.Next() {
		var s Swim
		errScan := rows.Scan(&s.Date, &s.DistanceM, &s.Assessment)
		if errScan != nil {
			return nil, errScan
		}

		swims = append(swims, &s)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return swims, nil
}

func (sw *swimModel) Summarize() *SwimSummary {
	summary := &SwimSummary{}

	swims, err := sw.GetAll()
	if err != nil {
		return summary
	}

	for _, swim := range swims {
		summary.TotalDistance += swim.DistanceM
		summary.TotalCount++

		year, week := swim.Date.ISOWeek()
		currentYear, currentWeek := time.Now().ISOWeek()
		if week == currentWeek && year == currentYear {
			summary.WeeklyDistance += swim.DistanceM
			summary.WeeklyCount++
		}

		month := swim.Date.Month()
		currentMonth := time.Now().Month()
		if month == currentMonth && year == currentYear {
			summary.MonthlyDistance += swim.DistanceM
			summary.MonthlyCount++
		}
	}

	return summary
}

func (sw *swimModel) Insert(date time.Time, distanceM int, assessment int) error {
	stmt := `INSERT INTO tracks_track (date, distance_m, assessment, user_id) VALUES ($1, $2, $3, $4);`

	_, err := sw.DB.Exec(stmt, date, distanceM, assessment, 1)
	if err != nil {
		return err
	}

	return nil
}

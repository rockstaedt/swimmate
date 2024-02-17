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

type Summary struct {
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
	Summarize([]*Swim) *Summary
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

func (sw *swimModel) Summarize(swims []*Swim) *Summary {
	var totalDistance int
	var totalCount int
	var monthlyDistance int
	var monthlyCount int
	var weeklyDistance int
	var weeklyCount int

	for _, swim := range swims {
		totalDistance += swim.DistanceM
		totalCount++

		if swim.Date.After(time.Now().AddDate(0, -1, 0)) {
			monthlyDistance += swim.DistanceM
			monthlyCount++
		}

		if swim.Date.After(time.Now().AddDate(0, 0, -7)) {
			weeklyDistance += swim.DistanceM
			weeklyCount++
		}
	}

	return &Summary{
		TotalDistance:   totalDistance,
		TotalCount:      totalCount,
		MonthlyDistance: monthlyDistance,
		MonthlyCount:    monthlyCount,
		WeeklyDistance:  weeklyDistance,
		WeeklyCount:     weeklyCount,
	}
}

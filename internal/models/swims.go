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
	YearMap         map[int]YearMap
}

type YearMap struct {
	SwimFigures
	MonthMap map[time.Month]SwimFigures
}

type SwimFigures struct {
	Count     int
	DistanceM int
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
	stmt := `SELECT date, distance_m, assessment FROM swims ORDER BY date ASC LIMIT 1;`

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
	stmt := `SELECT date, distance_m, assessment FROM swims WHERE user_id = $1 ORDER BY date ASC;`

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
	summary := &SwimSummary{YearMap: make(map[int]YearMap)}

	swims, err := sw.GetAll()
	if err != nil {
		return summary
	}

	for _, swim := range swims {
		summary.pushYearlyFigures(swim)
		summary.pushWeeklyFigures(swim)

		summary.updateYearMap(swim)
		summary.updateMonthMap(swim)
	}

	summary.MonthlyDistance = summary.YearMap[time.Now().Year()].MonthMap[time.Now().Month()].DistanceM
	summary.MonthlyCount = summary.YearMap[time.Now().Year()].MonthMap[time.Now().Month()].Count

	return summary
}

func (sw *swimModel) Insert(date time.Time, distanceM int, assessment int) error {
	stmt := `INSERT INTO swims (date, distance_m, assessment, user_id) VALUES ($1, $2, $3, $4);`

	_, err := sw.DB.Exec(stmt, date, distanceM, assessment, 1)
	if err != nil {
		return err
	}

	return nil
}

func (s *SwimSummary) pushYearlyFigures(swim *Swim) {
	s.TotalDistance += swim.DistanceM
	s.TotalCount++
}

func (s *SwimSummary) pushWeeklyFigures(swim *Swim) {
	year, week := swim.Date.ISOWeek()
	currentYear, currentWeek := time.Now().ISOWeek()

	if week == currentWeek && year == currentYear {
		s.WeeklyDistance += swim.DistanceM
		s.WeeklyCount++
	}
}

func (s *SwimSummary) updateYearMap(swim *Swim) {
	year := swim.Date.Year()

	yearMap, ok := s.YearMap[year]
	if !ok {
		yearMap = YearMap{}
		yearMap.MonthMap = make(map[time.Month]SwimFigures)
		for i := 1; i <= 12; i++ {
			yearMap.MonthMap[time.Month(i)] = SwimFigures{Count: 0, DistanceM: 0}
		}
	}

	yearMap.Count++
	yearMap.DistanceM += swim.DistanceM

	s.YearMap[year] = yearMap
}

func (s *SwimSummary) updateMonthMap(swim *Swim) {
	month := swim.Date.Month()
	yearMap := s.YearMap[swim.Date.Year()]
	monthMap := yearMap.MonthMap[month]

	monthMap.Count++
	monthMap.DistanceM += swim.DistanceM

	yearMap.MonthMap[month] = monthMap

}

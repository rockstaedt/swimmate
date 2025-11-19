package models

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	SwimSortDate       = "date"
	SwimSortDistance   = "distance"
	SwimSortAssessment = "assessment"

	SortDirectionAsc  = "asc"
	SortDirectionDesc = "desc"
)

type Swim struct {
	Id         int
	Date       time.Time
	DistanceM  int
	Assessment int
}

type SwimSummary struct {
	TotalDistance    int
	TotalCount       int
	MonthlyDistance  int
	MonthlyCount     int
	WeeklyDistance   int
	WeeklyCount      int
	MaxActivityCount int
	YearMap          map[int]YearMap
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
	GetAll(userId int) ([]*Swim, error)
	GetPaginated(userId int, limit int, offset int, sort string, direction string) ([]*Swim, error)
	Insert(date time.Time, distanceM int, assessment int, userId int) error
	Summarize(userId int) *SwimSummary
}

type swimModel struct {
	DB *sql.DB
}

func NewSwimModel(db *sql.DB) SwimModel {
	return &swimModel{DB: db}
}

func (sw *swimModel) Get() (*Swim, error) {
	stmt := `SELECT id, date, distance_m, assessment FROM swims ORDER BY date ASC LIMIT 1;`

	row := sw.DB.QueryRow(stmt)

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

func (sw *swimModel) GetAll(userId int) ([]*Swim, error) {
	stmt := `SELECT id, date, distance_m, assessment FROM swims WHERE user_id = $1 ORDER BY date ASC;`

	rows, err := sw.DB.Query(stmt, userId)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := rows.Close(); err != nil {
			// Ignore Close error to avoid overriding return error
			_ = err
		}
	}()

	var swims []*Swim
	for rows.Next() {
		var s Swim
		errScan := rows.Scan(&s.Id, &s.Date, &s.DistanceM, &s.Assessment)
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

func (sw *swimModel) Summarize(userId int) *SwimSummary {
	summary := &SwimSummary{YearMap: make(map[int]YearMap)}

	swims, err := sw.GetAll(userId)
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

	// Calculate max activity count for chart scaling
	summary.MaxActivityCount = summary.MonthlyCount
	if summary.WeeklyCount > summary.MaxActivityCount {
		summary.MaxActivityCount = summary.WeeklyCount
	}
	if summary.MaxActivityCount == 0 {
		summary.MaxActivityCount = 10
	}

	return summary
}

func (sw *swimModel) GetPaginated(userId int, limit int, offset int, sort string, direction string) ([]*Swim, error) {
	sortColumn := sanitizeSortColumn(sort)
	sortDirection := sanitizeSortDirection(direction)

	stmt := fmt.Sprintf(
		`SELECT id, date, distance_m, assessment FROM swims WHERE user_id = $1 ORDER BY %s %s LIMIT $2 OFFSET $3;`,
		sortColumn,
		sortDirection,
	)

	rows, err := sw.DB.Query(stmt, userId, limit, offset)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := rows.Close(); err != nil {
			// Ignore Close error to avoid overriding return error
			_ = err
		}
	}()

	var swims []*Swim
	for rows.Next() {
		var s Swim
		errScan := rows.Scan(&s.Id, &s.Date, &s.DistanceM, &s.Assessment)
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

func (sw *swimModel) Insert(date time.Time, distanceM int, assessment int, userId int) error {
	stmt := `INSERT INTO swims (date, distance_m, assessment, user_id) VALUES ($1, $2, $3, $4);`

	_, err := sw.DB.Exec(stmt, date, distanceM, assessment, userId)
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

var sortColumnMap = map[string]string{
	SwimSortDate:       "date",
	SwimSortDistance:   "distance_m",
	SwimSortAssessment: "assessment",
}

func sanitizeSortColumn(sort string) string {
	if column, ok := sortColumnMap[sort]; ok {
		return column
	}
	return sortColumnMap[SwimSortDate]
}

func sanitizeSortDirection(direction string) string {
	if strings.EqualFold(direction, SortDirectionAsc) {
		return "ASC"
	}
	return "DESC"
}

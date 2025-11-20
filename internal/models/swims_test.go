package models

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestSwimModelInsert(t *testing.T) {
	tests := []struct {
		name        string
		date        time.Time
		distanceM   int
		assessment  int
		userId      int
		setupMock   func(mock sqlmock.Sqlmock)
		expectError bool
		errorMsg    string
	}{
		{
			name:       "successful insert",
			date:       time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			distanceM:  1000,
			assessment: 2,
			userId:     1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO swims").
					WithArgs(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), 1000, 2, 1).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectError: false,
		},
		{
			name:       "database error on insert",
			date:       time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			distanceM:  1000,
			assessment: 2,
			userId:     1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO swims").
					WithArgs(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), 1000, 2, 1).
					WillReturnError(errors.New("database connection lost"))
			},
			expectError: true,
			errorMsg:    "database connection lost",
		},
		{
			name:       "insert with zero distance",
			date:       time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			distanceM:  0,
			assessment: 0,
			userId:     1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO swims").
					WithArgs(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), 0, 0, 1).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectError: false,
		},
		{
			name:       "insert with large distance",
			date:       time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			distanceM:  10000,
			assessment: 2,
			userId:     99,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO swims").
					WithArgs(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), 10000, 2, 99).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer func() {
				_ = db.Close()
			}()

			tt.setupMock(mock)

			model := NewSwimModel(db)
			err = model.Insert(tt.date, tt.distanceM, tt.assessment, tt.userId)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSwimModelUpdate(t *testing.T) {
	tests := []struct {
		name        string
		userId      int
		swimId      int
		date        time.Time
		distanceM   int
		assessment  int
		setupMock   func(mock sqlmock.Sqlmock)
		expectError bool
		errorType   error
	}{
		{
			name:       "successful update",
			userId:     1,
			swimId:     10,
			date:       time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			distanceM:  2000,
			assessment: 2,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE swims SET date = \\$1, distance_m = \\$2, assessment = \\$3 WHERE id = \\$4 AND user_id = \\$5").
					WithArgs(time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), 2000, 2, 10, 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name:       "no rows updated",
			userId:     1,
			swimId:     999,
			date:       time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			distanceM:  1000,
			assessment: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE swims SET date = \\$1, distance_m = \\$2, assessment = \\$3 WHERE id = \\$4 AND user_id = \\$5").
					WithArgs(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), 1000, 1, 999, 1).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectError: true,
			errorType:   ErrNoRecord,
		},
		{
			name:       "database error",
			userId:     1,
			swimId:     5,
			date:       time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
			distanceM:  1500,
			assessment: 0,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE swims SET date = \\$1, distance_m = \\$2, assessment = \\$3 WHERE id = \\$4 AND user_id = \\$5").
					WithArgs(time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC), 1500, 0, 5, 1).
					WillReturnError(errors.New("update failed"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer func() {
				_ = db.Close()
			}()

			tt.setupMock(mock)

			model := NewSwimModel(db)
			err = model.Update(tt.swimId, tt.userId, tt.date, tt.distanceM, tt.assessment)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSwimModelGet(t *testing.T) {
	tests := []struct {
		name         string
		setupMock    func(mock sqlmock.Sqlmock)
		expectError  bool
		expectedSwim *Swim
		errorType    error
	}{
		{
			name: "successful get earliest swim",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "date", "distance_m", "assessment"}).
					AddRow(1, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 1500, 2)
				mock.ExpectQuery("SELECT id, date, distance_m, assessment FROM swims ORDER BY date ASC LIMIT 1").
					WillReturnRows(rows)
			},
			expectError: false,
			expectedSwim: &Swim{
				Date:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				DistanceM:  1500,
				Assessment: 2,
			},
		},
		{
			name: "no records found",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, date, distance_m, assessment FROM swims ORDER BY date ASC LIMIT 1").
					WillReturnError(sql.ErrNoRows)
			},
			expectError:  true,
			expectedSwim: &Swim{},
			errorType:    ErrNoRecord,
		},
		{
			name: "database error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, date, distance_m, assessment FROM swims ORDER BY date ASC LIMIT 1").
					WillReturnError(errors.New("connection timeout"))
			},
			expectError:  true,
			expectedSwim: &Swim{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer func() {
				_ = db.Close()
			}()

			tt.setupMock(mock)

			model := NewSwimModel(db)
			swim, err := model.Get()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
				assert.Equal(t, tt.expectedSwim, swim)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedSwim.Date, swim.Date)
				assert.Equal(t, tt.expectedSwim.DistanceM, swim.DistanceM)
				assert.Equal(t, tt.expectedSwim.Assessment, swim.Assessment)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSwimModelGetByID(t *testing.T) {
	tests := []struct {
		name         string
		userId       int
		swimId       int
		setupMock    func(mock sqlmock.Sqlmock)
		expectError  bool
		expectedSwim *Swim
		errorType    error
	}{
		{
			name:   "successful fetch",
			userId: 1,
			swimId: 10,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "date", "distance_m", "assessment"}).
					AddRow(10, time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), 1800, 1)
				mock.ExpectQuery("SELECT id, date, distance_m, assessment FROM swims WHERE id = \\$1 AND user_id = \\$2").
					WithArgs(10, 1).
					WillReturnRows(rows)
			},
			expectedSwim: &Swim{
				Id:         10,
				Date:       time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				DistanceM:  1800,
				Assessment: 1,
			},
		},
		{
			name:   "not found",
			userId: 1,
			swimId: 999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, date, distance_m, assessment FROM swims WHERE id = \\$1 AND user_id = \\$2").
					WithArgs(999, 1).
					WillReturnError(sql.ErrNoRows)
			},
			expectError:  true,
			expectedSwim: &Swim{},
			errorType:    ErrNoRecord,
		},
		{
			name:   "database error",
			userId: 1,
			swimId: 5,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, date, distance_m, assessment FROM swims WHERE id = \\$1 AND user_id = \\$2").
					WithArgs(5, 1).
					WillReturnError(errors.New("query failed"))
			},
			expectError:  true,
			expectedSwim: &Swim{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer func() {
				_ = db.Close()
			}()

			tt.setupMock(mock)

			model := NewSwimModel(db)
			swim, err := model.GetByID(tt.userId, tt.swimId)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
				assert.Equal(t, tt.expectedSwim, swim)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedSwim, swim)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSwimModelGetAll(t *testing.T) {
	tests := []struct {
		name          string
		userId        int
		setupMock     func(mock sqlmock.Sqlmock)
		expectError   bool
		expectedSwims []*Swim
		errorMsg      string
	}{
		{
			name:   "successful get all swims",
			userId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "date", "distance_m", "assessment"}).
					AddRow(1, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 1000, 1).
					AddRow(2, time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), 1500, 2).
					AddRow(3, time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), 2000, 2)
				mock.ExpectQuery("SELECT id, date, distance_m, assessment FROM swims WHERE user_id = \\$1 ORDER BY date ASC").
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectError: false,
			expectedSwims: []*Swim{
				{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), DistanceM: 1000, Assessment: 1},
				{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), DistanceM: 1500, Assessment: 2},
				{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), DistanceM: 2000, Assessment: 2},
			},
		},
		{
			name:   "empty result set",
			userId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "date", "distance_m", "assessment"})
				mock.ExpectQuery("SELECT id, date, distance_m, assessment FROM swims WHERE user_id = \\$1 ORDER BY date ASC").
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectError:   false,
			expectedSwims: nil,
		},
		{
			name:   "database error on query",
			userId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, date, distance_m, assessment FROM swims WHERE user_id = \\$1 ORDER BY date ASC").
					WithArgs(1).
					WillReturnError(errors.New("query execution failed"))
			},
			expectError:   true,
			expectedSwims: nil,
			errorMsg:      "query execution failed",
		},
		{
			name:   "scan error on rows",
			userId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "date", "distance_m", "assessment"}).
					AddRow(1, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 1000, 1).
					AddRow(2, "invalid-date", 1500, 2) // Invalid date will cause scan error
				mock.ExpectQuery("SELECT id, date, distance_m, assessment FROM swims WHERE user_id = \\$1 ORDER BY date ASC").
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectError:   true,
			expectedSwims: nil,
		},
		{
			name:   "ordering verification - dates in ascending order",
			userId: 5,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "date", "distance_m", "assessment"}).
					AddRow(1, time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC), 500, 1).
					AddRow(2, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), 750, 2)
				mock.ExpectQuery("SELECT id, date, distance_m, assessment FROM swims WHERE user_id = \\$1 ORDER BY date ASC").
					WithArgs(5).
					WillReturnRows(rows)
			},
			expectError: false,
			expectedSwims: []*Swim{
				{Date: time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC), DistanceM: 500, Assessment: 1},
				{Date: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), DistanceM: 750, Assessment: 2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer func() {
				_ = db.Close()
			}()

			tt.setupMock(mock)

			model := NewSwimModel(db)
			swims, err := model.GetAll(tt.userId)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.expectedSwims), len(swims))
				for i, expectedSwim := range tt.expectedSwims {
					assert.Equal(t, expectedSwim.Date, swims[i].Date)
					assert.Equal(t, expectedSwim.DistanceM, swims[i].DistanceM)
					assert.Equal(t, expectedSwim.Assessment, swims[i].Assessment)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSwimModelGetPaginated(t *testing.T) {
	tests := []struct {
		name          string
		userId        int
		limit         int
		offset        int
		sort          string
		direction     string
		setupMock     func(mock sqlmock.Sqlmock)
		expectError   bool
		expectedSwims []*Swim
		errorMsg      string
	}{
		{
			name:      "successful pagination - first page",
			userId:    1,
			limit:     2,
			offset:    0,
			sort:      SwimSortDate,
			direction: SortDirectionDesc,
			setupMock: func(mock sqlmock.Sqlmock) {
				query := regexp.QuoteMeta(fmt.Sprintf(
					"SELECT id, date, distance_m, assessment FROM swims WHERE user_id = $1 ORDER BY %s %s LIMIT $2 OFFSET $3",
					sortColumnMap[SwimSortDate],
					"DESC",
				))
				rows := sqlmock.NewRows([]string{"id", "date", "distance_m", "assessment"}).
					AddRow(1, time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), 2000, 2).
					AddRow(2, time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), 1500, 2)
				mock.ExpectQuery(query).
					WithArgs(1, 2, 0).
					WillReturnRows(rows)
			},
			expectError: false,
			expectedSwims: []*Swim{
				{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), DistanceM: 2000, Assessment: 2},
				{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), DistanceM: 1500, Assessment: 2},
			},
		},
		{
			name:      "successful pagination - second page",
			userId:    1,
			limit:     2,
			offset:    2,
			sort:      SwimSortDate,
			direction: SortDirectionDesc,
			setupMock: func(mock sqlmock.Sqlmock) {
				query := regexp.QuoteMeta(fmt.Sprintf(
					"SELECT id, date, distance_m, assessment FROM swims WHERE user_id = $1 ORDER BY %s %s LIMIT $2 OFFSET $3",
					sortColumnMap[SwimSortDate],
					"DESC",
				))
				rows := sqlmock.NewRows([]string{"id", "date", "distance_m", "assessment"}).
					AddRow(3, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 1000, 1)
				mock.ExpectQuery(query).
					WithArgs(1, 2, 2).
					WillReturnRows(rows)
			},
			expectError: false,
			expectedSwims: []*Swim{
				{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), DistanceM: 1000, Assessment: 1},
			},
		},
		{
			name:      "empty result - offset beyond records",
			userId:    1,
			limit:     20,
			offset:    100,
			sort:      SwimSortDate,
			direction: SortDirectionDesc,
			setupMock: func(mock sqlmock.Sqlmock) {
				query := regexp.QuoteMeta(fmt.Sprintf(
					"SELECT id, date, distance_m, assessment FROM swims WHERE user_id = $1 ORDER BY %s %s LIMIT $2 OFFSET $3",
					sortColumnMap[SwimSortDate],
					"DESC",
				))
				rows := sqlmock.NewRows([]string{"id", "date", "distance_m", "assessment"})
				mock.ExpectQuery(query).
					WithArgs(1, 20, 100).
					WillReturnRows(rows)
			},
			expectError:   false,
			expectedSwims: nil,
		},
		{
			name:      "exact multiple of page size",
			userId:    1,
			limit:     20,
			offset:    0,
			sort:      SwimSortDate,
			direction: SortDirectionDesc,
			setupMock: func(mock sqlmock.Sqlmock) {
				query := regexp.QuoteMeta(fmt.Sprintf(
					"SELECT id, date, distance_m, assessment FROM swims WHERE user_id = $1 ORDER BY %s %s LIMIT $2 OFFSET $3",
					sortColumnMap[SwimSortDate],
					"DESC",
				))
				// Simulate exactly 20 records
				rows := sqlmock.NewRows([]string{"id", "date", "distance_m", "assessment"})
				for i := 20; i > 0; i-- {
					rows.AddRow(i, time.Date(2024, 1, i, 0, 0, 0, 0, time.UTC), 1000*i, 2)
				}
				mock.ExpectQuery(query).
					WithArgs(1, 20, 0).
					WillReturnRows(rows)
			},
			expectError: false,
			expectedSwims: func() []*Swim {
				swims := make([]*Swim, 20)
				for i := 0; i < 20; i++ {
					swims[i] = &Swim{
						Date:       time.Date(2024, 1, 20-i, 0, 0, 0, 0, time.UTC),
						DistanceM:  1000 * (20 - i),
						Assessment: 2,
					}
				}
				return swims
			}(),
		},
		{
			name:      "database error on paginated query",
			userId:    1,
			limit:     20,
			offset:    0,
			sort:      SwimSortDate,
			direction: SortDirectionDesc,
			setupMock: func(mock sqlmock.Sqlmock) {
				query := regexp.QuoteMeta(fmt.Sprintf(
					"SELECT id, date, distance_m, assessment FROM swims WHERE user_id = $1 ORDER BY %s %s LIMIT $2 OFFSET $3",
					sortColumnMap[SwimSortDate],
					"DESC",
				))
				mock.ExpectQuery(query).
					WithArgs(1, 20, 0).
					WillReturnError(errors.New("pagination query failed"))
			},
			expectError:   true,
			expectedSwims: nil,
			errorMsg:      "pagination query failed",
		},
		{
			name:      "ordering verification - DESC order",
			userId:    2,
			limit:     3,
			offset:    0,
			sort:      SwimSortDate,
			direction: SortDirectionDesc,
			setupMock: func(mock sqlmock.Sqlmock) {
				query := regexp.QuoteMeta(fmt.Sprintf(
					"SELECT id, date, distance_m, assessment FROM swims WHERE user_id = $1 ORDER BY %s %s LIMIT $2 OFFSET $3",
					sortColumnMap[SwimSortDate],
					"DESC",
				))
				rows := sqlmock.NewRows([]string{"id", "date", "distance_m", "assessment"}).
					AddRow(1, time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC), 3000, 2).
					AddRow(2, time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC), 2000, 2).
					AddRow(3, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 1000, 1)
				mock.ExpectQuery(query).
					WithArgs(2, 3, 0).
					WillReturnRows(rows)
			},
			expectError: false,
			expectedSwims: []*Swim{
				{Date: time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC), DistanceM: 3000, Assessment: 2},
				{Date: time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC), DistanceM: 2000, Assessment: 2},
				{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), DistanceM: 1000, Assessment: 1},
			},
		},
		{
			name:      "sorting by distance ascending",
			userId:    1,
			limit:     5,
			offset:    0,
			sort:      SwimSortDistance,
			direction: SortDirectionAsc,
			setupMock: func(mock sqlmock.Sqlmock) {
				query := regexp.QuoteMeta(fmt.Sprintf(
					"SELECT id, date, distance_m, assessment FROM swims WHERE user_id = $1 ORDER BY %s %s LIMIT $2 OFFSET $3",
					sortColumnMap[SwimSortDistance],
					"ASC",
				))
				rows := sqlmock.NewRows([]string{"id", "date", "distance_m", "assessment"})
				mock.ExpectQuery(query).
					WithArgs(1, 5, 0).
					WillReturnRows(rows)
			},
		},
		{
			name:      "invalid sort and direction fall back to defaults",
			userId:    1,
			limit:     5,
			offset:    0,
			sort:      "invalid",
			direction: "weird",
			setupMock: func(mock sqlmock.Sqlmock) {
				query := regexp.QuoteMeta(fmt.Sprintf(
					"SELECT id, date, distance_m, assessment FROM swims WHERE user_id = $1 ORDER BY %s %s LIMIT $2 OFFSET $3",
					sortColumnMap[SwimSortDate],
					"DESC",
				))
				rows := sqlmock.NewRows([]string{"id", "date", "distance_m", "assessment"})
				mock.ExpectQuery(query).
					WithArgs(1, 5, 0).
					WillReturnRows(rows)
			},
		},
		{
			name:      "uppercase direction still treated as ascending",
			userId:    1,
			limit:     5,
			offset:    0,
			sort:      SwimSortDate,
			direction: strings.ToUpper(SortDirectionAsc),
			setupMock: func(mock sqlmock.Sqlmock) {
				query := regexp.QuoteMeta(fmt.Sprintf(
					"SELECT id, date, distance_m, assessment FROM swims WHERE user_id = $1 ORDER BY %s %s LIMIT $2 OFFSET $3",
					sortColumnMap[SwimSortDate],
					"ASC",
				))
				rows := sqlmock.NewRows([]string{"id", "date", "distance_m", "assessment"})
				mock.ExpectQuery(query).
					WithArgs(1, 5, 0).
					WillReturnRows(rows)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer func() {
				_ = db.Close()
			}()

			tt.setupMock(mock)

			model := NewSwimModel(db)
			swims, err := model.GetPaginated(tt.userId, tt.limit, tt.offset, tt.sort, tt.direction)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.expectedSwims), len(swims))
				for i, expectedSwim := range tt.expectedSwims {
					assert.Equal(t, expectedSwim.Date, swims[i].Date)
					assert.Equal(t, expectedSwim.DistanceM, swims[i].DistanceM)
					assert.Equal(t, expectedSwim.Assessment, swims[i].Assessment)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSwimModelSummarize(t *testing.T) {
	tests := []struct {
		name            string
		userId          int
		setupMock       func(mock sqlmock.Sqlmock)
		expectedSummary *SwimSummary
	}{
		{
			name:   "empty dataset",
			userId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "date", "distance_m", "assessment"})
				mock.ExpectQuery("SELECT id, date, distance_m, assessment FROM swims WHERE user_id = \\$1 ORDER BY date ASC").
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedSummary: &SwimSummary{
				TotalDistance:   0,
				TotalCount:      0,
				MonthlyDistance: 0,
				MonthlyCount:    0,
				WeeklyDistance:  0,
				WeeklyCount:     0,
				YearMap:         make(map[int]YearMap),
			},
		},
		{
			name:   "single swim from past year",
			userId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "date", "distance_m", "assessment"}).
					AddRow(1, time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC), 1500, 2)
				mock.ExpectQuery("SELECT id, date, distance_m, assessment FROM swims WHERE user_id = \\$1 ORDER BY date ASC").
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedSummary: &SwimSummary{
				TotalDistance:   1500,
				TotalCount:      1,
				MonthlyDistance: 0,
				MonthlyCount:    0,
				WeeklyDistance:  0,
				WeeklyCount:     0,
				YearMap: map[int]YearMap{
					2020: {
						SwimFigures: SwimFigures{Count: 1, DistanceM: 1500},
						MonthMap: map[time.Month]SwimFigures{
							time.January:   {Count: 0, DistanceM: 0},
							time.February:  {Count: 0, DistanceM: 0},
							time.March:     {Count: 0, DistanceM: 0},
							time.April:     {Count: 0, DistanceM: 0},
							time.May:       {Count: 0, DistanceM: 0},
							time.June:      {Count: 1, DistanceM: 1500},
							time.July:      {Count: 0, DistanceM: 0},
							time.August:    {Count: 0, DistanceM: 0},
							time.September: {Count: 0, DistanceM: 0},
							time.October:   {Count: 0, DistanceM: 0},
							time.November:  {Count: 0, DistanceM: 0},
							time.December:  {Count: 0, DistanceM: 0},
						},
					},
				},
			},
		},
		{
			name:   "multiple swims across different months in same year",
			userId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "date", "distance_m", "assessment"}).
					AddRow(1, time.Date(2020, 1, 10, 0, 0, 0, 0, time.UTC), 1000, 1).
					AddRow(2, time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC), 1500, 2).
					AddRow(3, time.Date(2020, 3, 20, 0, 0, 0, 0, time.UTC), 2000, 2)
				mock.ExpectQuery("SELECT id, date, distance_m, assessment FROM swims WHERE user_id = \\$1 ORDER BY date ASC").
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedSummary: &SwimSummary{
				TotalDistance:   4500,
				TotalCount:      3,
				MonthlyDistance: 0,
				MonthlyCount:    0,
				WeeklyDistance:  0,
				WeeklyCount:     0,
				YearMap: map[int]YearMap{
					2020: {
						SwimFigures: SwimFigures{Count: 3, DistanceM: 4500},
						MonthMap: map[time.Month]SwimFigures{
							time.January:   {Count: 1, DistanceM: 1000},
							time.February:  {Count: 0, DistanceM: 0},
							time.March:     {Count: 2, DistanceM: 3500},
							time.April:     {Count: 0, DistanceM: 0},
							time.May:       {Count: 0, DistanceM: 0},
							time.June:      {Count: 0, DistanceM: 0},
							time.July:      {Count: 0, DistanceM: 0},
							time.August:    {Count: 0, DistanceM: 0},
							time.September: {Count: 0, DistanceM: 0},
							time.October:   {Count: 0, DistanceM: 0},
							time.November:  {Count: 0, DistanceM: 0},
							time.December:  {Count: 0, DistanceM: 0},
						},
					},
				},
			},
		},
		{
			name:   "multiple swims across different years",
			userId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "date", "distance_m", "assessment"}).
					AddRow(1, time.Date(2019, 12, 25, 0, 0, 0, 0, time.UTC), 1000, 1).
					AddRow(2, time.Date(2020, 1, 5, 0, 0, 0, 0, time.UTC), 1500, 2).
					AddRow(3, time.Date(2021, 6, 15, 0, 0, 0, 0, time.UTC), 2000, 2)
				mock.ExpectQuery("SELECT id, date, distance_m, assessment FROM swims WHERE user_id = \\$1 ORDER BY date ASC").
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedSummary: &SwimSummary{
				TotalDistance:   4500,
				TotalCount:      3,
				MonthlyDistance: 0,
				MonthlyCount:    0,
				WeeklyDistance:  0,
				WeeklyCount:     0,
				YearMap: map[int]YearMap{
					2019: {
						SwimFigures: SwimFigures{Count: 1, DistanceM: 1000},
						MonthMap: map[time.Month]SwimFigures{
							time.January:   {Count: 0, DistanceM: 0},
							time.February:  {Count: 0, DistanceM: 0},
							time.March:     {Count: 0, DistanceM: 0},
							time.April:     {Count: 0, DistanceM: 0},
							time.May:       {Count: 0, DistanceM: 0},
							time.June:      {Count: 0, DistanceM: 0},
							time.July:      {Count: 0, DistanceM: 0},
							time.August:    {Count: 0, DistanceM: 0},
							time.September: {Count: 0, DistanceM: 0},
							time.October:   {Count: 0, DistanceM: 0},
							time.November:  {Count: 0, DistanceM: 0},
							time.December:  {Count: 1, DistanceM: 1000},
						},
					},
					2020: {
						SwimFigures: SwimFigures{Count: 1, DistanceM: 1500},
						MonthMap: map[time.Month]SwimFigures{
							time.January:   {Count: 1, DistanceM: 1500},
							time.February:  {Count: 0, DistanceM: 0},
							time.March:     {Count: 0, DistanceM: 0},
							time.April:     {Count: 0, DistanceM: 0},
							time.May:       {Count: 0, DistanceM: 0},
							time.June:      {Count: 0, DistanceM: 0},
							time.July:      {Count: 0, DistanceM: 0},
							time.August:    {Count: 0, DistanceM: 0},
							time.September: {Count: 0, DistanceM: 0},
							time.October:   {Count: 0, DistanceM: 0},
							time.November:  {Count: 0, DistanceM: 0},
							time.December:  {Count: 0, DistanceM: 0},
						},
					},
					2021: {
						SwimFigures: SwimFigures{Count: 1, DistanceM: 2000},
						MonthMap: map[time.Month]SwimFigures{
							time.January:   {Count: 0, DistanceM: 0},
							time.February:  {Count: 0, DistanceM: 0},
							time.March:     {Count: 0, DistanceM: 0},
							time.April:     {Count: 0, DistanceM: 0},
							time.May:       {Count: 0, DistanceM: 0},
							time.June:      {Count: 1, DistanceM: 2000},
							time.July:      {Count: 0, DistanceM: 0},
							time.August:    {Count: 0, DistanceM: 0},
							time.September: {Count: 0, DistanceM: 0},
							time.October:   {Count: 0, DistanceM: 0},
							time.November:  {Count: 0, DistanceM: 0},
							time.December:  {Count: 0, DistanceM: 0},
						},
					},
				},
			},
		},
		{
			name:   "current week calculation",
			userId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				now := time.Now()
				rows := sqlmock.NewRows([]string{"id", "date", "distance_m", "assessment"}).
					AddRow(1, now, 1500, 2)
				mock.ExpectQuery("SELECT id, date, distance_m, assessment FROM swims WHERE user_id = \\$1 ORDER BY date ASC").
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedSummary: &SwimSummary{
				TotalDistance:   1500,
				TotalCount:      1,
				MonthlyDistance: 1500,
				MonthlyCount:    1,
				WeeklyDistance:  1500,
				WeeklyCount:     1,
				YearMap: map[int]YearMap{
					time.Now().Year(): {
						SwimFigures: SwimFigures{Count: 1, DistanceM: 1500},
						MonthMap: func() map[time.Month]SwimFigures {
							monthMap := make(map[time.Month]SwimFigures)
							for i := 1; i <= 12; i++ {
								monthMap[time.Month(i)] = SwimFigures{Count: 0, DistanceM: 0}
							}
							monthMap[time.Now().Month()] = SwimFigures{Count: 1, DistanceM: 1500}
							return monthMap
						}(),
					},
				},
			},
		},
		{
			name:   "database error returns empty summary",
			userId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, date, distance_m, assessment FROM swims WHERE user_id = \\$1 ORDER BY date ASC").
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			expectedSummary: &SwimSummary{
				TotalDistance:   0,
				TotalCount:      0,
				MonthlyDistance: 0,
				MonthlyCount:    0,
				WeeklyDistance:  0,
				WeeklyCount:     0,
				YearMap:         make(map[int]YearMap),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer func() {
				_ = db.Close()
			}()

			tt.setupMock(mock)

			model := NewSwimModel(db)
			summary := model.Summarize(tt.userId)

			assert.Equal(t, tt.expectedSummary.TotalDistance, summary.TotalDistance)
			assert.Equal(t, tt.expectedSummary.TotalCount, summary.TotalCount)
			assert.Equal(t, tt.expectedSummary.MonthlyDistance, summary.MonthlyDistance)
			assert.Equal(t, tt.expectedSummary.MonthlyCount, summary.MonthlyCount)
			assert.Equal(t, tt.expectedSummary.WeeklyDistance, summary.WeeklyDistance)
			assert.Equal(t, tt.expectedSummary.WeeklyCount, summary.WeeklyCount)

			// Verify YearMap
			assert.Equal(t, len(tt.expectedSummary.YearMap), len(summary.YearMap))
			for year, expectedYearMap := range tt.expectedSummary.YearMap {
				actualYearMap, exists := summary.YearMap[year]
				assert.True(t, exists, "Year %d should exist in YearMap", year)
				assert.Equal(t, expectedYearMap.Count, actualYearMap.Count)
				assert.Equal(t, expectedYearMap.DistanceM, actualYearMap.DistanceM)

				// Verify all 12 months are initialized
				assert.Equal(t, 12, len(actualYearMap.MonthMap))
				for month := time.January; month <= time.December; month++ {
					expectedMonth := expectedYearMap.MonthMap[month]
					actualMonth := actualYearMap.MonthMap[month]
					assert.Equal(t, expectedMonth.Count, actualMonth.Count, "Month %s count mismatch", month)
					assert.Equal(t, expectedMonth.DistanceM, actualMonth.DistanceM, "Month %s distance mismatch", month)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSwimSummaryHelperMethods(t *testing.T) {
	t.Run("pushYearlyFigures", func(t *testing.T) {
		summary := &SwimSummary{}
		swim := &Swim{DistanceM: 1500}

		summary.pushYearlyFigures(swim)
		assert.Equal(t, 1500, summary.TotalDistance)
		assert.Equal(t, 1, summary.TotalCount)

		summary.pushYearlyFigures(swim)
		assert.Equal(t, 3000, summary.TotalDistance)
		assert.Equal(t, 2, summary.TotalCount)
	})

	t.Run("pushWeeklyFigures - current week", func(t *testing.T) {
		summary := &SwimSummary{}
		now := time.Now()
		swim := &Swim{Date: now, DistanceM: 2000}

		summary.pushWeeklyFigures(swim)
		assert.Equal(t, 2000, summary.WeeklyDistance)
		assert.Equal(t, 1, summary.WeeklyCount)
	})

	t.Run("pushWeeklyFigures - past week", func(t *testing.T) {
		summary := &SwimSummary{}
		pastDate := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		swim := &Swim{Date: pastDate, DistanceM: 2000}

		summary.pushWeeklyFigures(swim)
		assert.Equal(t, 0, summary.WeeklyDistance)
		assert.Equal(t, 0, summary.WeeklyCount)
	})

	t.Run("updateYearMap - creates new year", func(t *testing.T) {
		summary := &SwimSummary{YearMap: make(map[int]YearMap)}
		swim := &Swim{Date: time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC), DistanceM: 1500}

		summary.updateYearMap(swim)

		yearMap, exists := summary.YearMap[2020]
		assert.True(t, exists)
		assert.Equal(t, 1, yearMap.Count)
		assert.Equal(t, 1500, yearMap.DistanceM)
		assert.Equal(t, 12, len(yearMap.MonthMap))
	})

	t.Run("updateYearMap - updates existing year", func(t *testing.T) {
		summary := &SwimSummary{YearMap: make(map[int]YearMap)}
		swim1 := &Swim{Date: time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC), DistanceM: 1500}
		swim2 := &Swim{Date: time.Date(2020, 7, 20, 0, 0, 0, 0, time.UTC), DistanceM: 2000}

		summary.updateYearMap(swim1)
		summary.updateYearMap(swim2)

		yearMap := summary.YearMap[2020]
		assert.Equal(t, 2, yearMap.Count)
		assert.Equal(t, 3500, yearMap.DistanceM)
	})

	t.Run("updateMonthMap", func(t *testing.T) {
		summary := &SwimSummary{YearMap: make(map[int]YearMap)}
		swim := &Swim{Date: time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC), DistanceM: 1500}

		summary.updateYearMap(swim)
		summary.updateMonthMap(swim)

		monthMap := summary.YearMap[2020].MonthMap[time.June]
		assert.Equal(t, 1, monthMap.Count)
		assert.Equal(t, 1500, monthMap.DistanceM)

		swim2 := &Swim{Date: time.Date(2020, 6, 20, 0, 0, 0, 0, time.UTC), DistanceM: 1000}
		summary.updateYearMap(swim2)
		summary.updateMonthMap(swim2)

		monthMap = summary.YearMap[2020].MonthMap[time.June]
		assert.Equal(t, 2, monthMap.Count)
		assert.Equal(t, 2500, monthMap.DistanceM)
	})
}

func TestSwimModelDelete(t *testing.T) {
	tests := []struct {
		name        string
		swimId      int
		userId      int
		setupMock   func(mock sqlmock.Sqlmock)
		expectError bool
		errorType   error
	}{
		{
			name:   "successful delete",
			swimId: 5,
			userId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("DELETE FROM swims WHERE id = \\$1 AND user_id = \\$2").
					WithArgs(5, 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectError: false,
		},
		{
			name:   "swim not found",
			swimId: 999,
			userId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("DELETE FROM swims WHERE id = \\$1 AND user_id = \\$2").
					WithArgs(999, 1).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectError: true,
			errorType:   ErrNoRecord,
		},
		{
			name:   "swim belongs to different user",
			swimId: 5,
			userId: 2,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("DELETE FROM swims WHERE id = \\$1 AND user_id = \\$2").
					WithArgs(5, 2).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectError: true,
			errorType:   ErrNoRecord,
		},
		{
			name:   "database error on delete",
			swimId: 5,
			userId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("DELETE FROM swims WHERE id = \\$1 AND user_id = \\$2").
					WithArgs(5, 1).
					WillReturnError(errors.New("database connection lost"))
			},
			expectError: true,
		},
		{
			name:   "delete with zero swim ID",
			swimId: 0,
			userId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("DELETE FROM swims WHERE id = \\$1 AND user_id = \\$2").
					WithArgs(0, 1).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectError: true,
			errorType:   ErrNoRecord,
		},
		{
			name:   "delete with negative swim ID",
			swimId: -5,
			userId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("DELETE FROM swims WHERE id = \\$1 AND user_id = \\$2").
					WithArgs(-5, 1).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectError: true,
			errorType:   ErrNoRecord,
		},
		{
			name:   "delete with zero user ID",
			swimId: 5,
			userId: 0,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("DELETE FROM swims WHERE id = \\$1 AND user_id = \\$2").
					WithArgs(5, 0).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectError: true,
			errorType:   ErrNoRecord,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer func() {
				_ = db.Close()
			}()

			tt.setupMock(mock)

			model := NewSwimModel(db)
			err = model.Delete(tt.swimId, tt.userId)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

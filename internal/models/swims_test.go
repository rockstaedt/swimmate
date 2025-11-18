package models

import (
	"database/sql"
	"errors"
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
			defer db.Close()

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
				rows := sqlmock.NewRows([]string{"date", "distance_m", "assessment"}).
					AddRow(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 1500, 2)
				mock.ExpectQuery("SELECT date, distance_m, assessment FROM swims ORDER BY date ASC LIMIT 1").
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
				mock.ExpectQuery("SELECT date, distance_m, assessment FROM swims ORDER BY date ASC LIMIT 1").
					WillReturnError(sql.ErrNoRows)
			},
			expectError:  true,
			expectedSwim: &Swim{},
			errorType:    ErrNoRecord,
		},
		{
			name: "database error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT date, distance_m, assessment FROM swims ORDER BY date ASC LIMIT 1").
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
			defer db.Close()

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
				rows := sqlmock.NewRows([]string{"date", "distance_m", "assessment"}).
					AddRow(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 1000, 1).
					AddRow(time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), 1500, 2).
					AddRow(time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), 2000, 2)
				mock.ExpectQuery("SELECT date, distance_m, assessment FROM swims WHERE user_id = \\$1 ORDER BY date ASC").
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
				rows := sqlmock.NewRows([]string{"date", "distance_m", "assessment"})
				mock.ExpectQuery("SELECT date, distance_m, assessment FROM swims WHERE user_id = \\$1 ORDER BY date ASC").
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
				mock.ExpectQuery("SELECT date, distance_m, assessment FROM swims WHERE user_id = \\$1 ORDER BY date ASC").
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
				rows := sqlmock.NewRows([]string{"date", "distance_m", "assessment"}).
					AddRow(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 1000, 1).
					AddRow("invalid-date", 1500, 2) // Invalid date will cause scan error
				mock.ExpectQuery("SELECT date, distance_m, assessment FROM swims WHERE user_id = \\$1 ORDER BY date ASC").
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
				rows := sqlmock.NewRows([]string{"date", "distance_m", "assessment"}).
					AddRow(time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC), 500, 1).
					AddRow(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), 750, 2)
				mock.ExpectQuery("SELECT date, distance_m, assessment FROM swims WHERE user_id = \\$1 ORDER BY date ASC").
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
			defer db.Close()

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
		setupMock     func(mock sqlmock.Sqlmock)
		expectError   bool
		expectedSwims []*Swim
		errorMsg      string
	}{
		{
			name:   "successful pagination - first page",
			userId: 1,
			limit:  2,
			offset: 0,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"date", "distance_m", "assessment"}).
					AddRow(time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), 2000, 2).
					AddRow(time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), 1500, 2)
				mock.ExpectQuery("SELECT date, distance_m, assessment FROM swims WHERE user_id = \\$1 ORDER BY date DESC LIMIT \\$2 OFFSET \\$3").
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
			name:   "successful pagination - second page",
			userId: 1,
			limit:  2,
			offset: 2,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"date", "distance_m", "assessment"}).
					AddRow(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 1000, 1)
				mock.ExpectQuery("SELECT date, distance_m, assessment FROM swims WHERE user_id = \\$1 ORDER BY date DESC LIMIT \\$2 OFFSET \\$3").
					WithArgs(1, 2, 2).
					WillReturnRows(rows)
			},
			expectError: false,
			expectedSwims: []*Swim{
				{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), DistanceM: 1000, Assessment: 1},
			},
		},
		{
			name:   "empty result - offset beyond records",
			userId: 1,
			limit:  20,
			offset: 100,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"date", "distance_m", "assessment"})
				mock.ExpectQuery("SELECT date, distance_m, assessment FROM swims WHERE user_id = \\$1 ORDER BY date DESC LIMIT \\$2 OFFSET \\$3").
					WithArgs(1, 20, 100).
					WillReturnRows(rows)
			},
			expectError:   false,
			expectedSwims: nil,
		},
		{
			name:   "exact multiple of page size",
			userId: 1,
			limit:  20,
			offset: 0,
			setupMock: func(mock sqlmock.Sqlmock) {
				// Simulate exactly 20 records
				rows := sqlmock.NewRows([]string{"date", "distance_m", "assessment"})
				for i := 20; i > 0; i-- {
					rows.AddRow(time.Date(2024, 1, i, 0, 0, 0, 0, time.UTC), 1000*i, 2)
				}
				mock.ExpectQuery("SELECT date, distance_m, assessment FROM swims WHERE user_id = \\$1 ORDER BY date DESC LIMIT \\$2 OFFSET \\$3").
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
			name:   "database error on paginated query",
			userId: 1,
			limit:  20,
			offset: 0,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT date, distance_m, assessment FROM swims WHERE user_id = \\$1 ORDER BY date DESC LIMIT \\$2 OFFSET \\$3").
					WithArgs(1, 20, 0).
					WillReturnError(errors.New("pagination query failed"))
			},
			expectError:   true,
			expectedSwims: nil,
			errorMsg:      "pagination query failed",
		},
		{
			name:   "ordering verification - DESC order",
			userId: 2,
			limit:  3,
			offset: 0,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"date", "distance_m", "assessment"}).
					AddRow(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC), 3000, 2).
					AddRow(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC), 2000, 2).
					AddRow(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 1000, 1)
				mock.ExpectQuery("SELECT date, distance_m, assessment FROM swims WHERE user_id = \\$1 ORDER BY date DESC LIMIT \\$2 OFFSET \\$3").
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			tt.setupMock(mock)

			model := NewSwimModel(db)
			swims, err := model.GetPaginated(tt.userId, tt.limit, tt.offset)

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

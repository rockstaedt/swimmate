package models

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestUserModelAuthenticate(t *testing.T) {
	// Generate a known password hash for testing
	validPassword := "test123"
	validHash, err := bcrypt.GenerateFromPassword([]byte(validPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to generate test password hash: %v", err)
	}

	tests := []struct {
		name          string
		username      string
		password      string
		setupMock     func(mock sqlmock.Sqlmock)
		expectError   bool
		expectedID    int
		errorType     error
	}{
		{
			name:     "successful authentication",
			username: "testuser",
			password: validPassword,
			setupMock: func(mock sqlmock.Sqlmock) {
				// Expect SELECT query for user
				rows := sqlmock.NewRows([]string{"id", "password"}).
					AddRow(1, validHash)
				mock.ExpectQuery("SELECT id, password FROM users WHERE username = \\$1").
					WithArgs("testuser").
					WillReturnRows(rows)

				// Expect UPDATE query for last_login
				mock.ExpectExec("UPDATE users SET last_login = \\$1 WHERE id = \\$2").
					WithArgs(sqlmock.AnyArg(), 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectError: false,
			expectedID:  1,
		},
		{
			name:     "user not found",
			username: "nonexistent",
			password: "anypassword",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, password FROM users WHERE username = \\$1").
					WithArgs("nonexistent").
					WillReturnError(sql.ErrNoRows)
			},
			expectError: true,
			expectedID:  0,
			errorType:   ErrInvalidCredentials,
		},
		{
			name:     "invalid password",
			username: "testuser",
			password: "wrongpassword",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "password"}).
					AddRow(1, validHash)
				mock.ExpectQuery("SELECT id, password FROM users WHERE username = \\$1").
					WithArgs("testuser").
					WillReturnRows(rows)
			},
			expectError: true,
			expectedID:  0,
			errorType:   ErrInvalidCredentials,
		},
		{
			name:     "database error on select",
			username: "testuser",
			password: validPassword,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, password FROM users WHERE username = \\$1").
					WithArgs("testuser").
					WillReturnError(errors.New("database connection error"))
			},
			expectError: true,
			expectedID:  0,
		},
		{
			name:     "database error on update",
			username: "testuser",
			password: validPassword,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "password"}).
					AddRow(1, validHash)
				mock.ExpectQuery("SELECT id, password FROM users WHERE username = \\$1").
					WithArgs("testuser").
					WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET last_login = \\$1 WHERE id = \\$2").
					WithArgs(sqlmock.AnyArg(), 1).
					WillReturnError(errors.New("update failed"))
			},
			expectError: true,
			expectedID:  0,
		},
		{
			name:     "empty username",
			username: "",
			password: validPassword,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, password FROM users WHERE username = \\$1").
					WithArgs("").
					WillReturnError(sql.ErrNoRows)
			},
			expectError: true,
			expectedID:  0,
			errorType:   ErrInvalidCredentials,
		},
		{
			name:     "empty password",
			username: "testuser",
			password: "",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "password"}).
					AddRow(1, validHash)
				mock.ExpectQuery("SELECT id, password FROM users WHERE username = \\$1").
					WithArgs("testuser").
					WillReturnRows(rows)
			},
			expectError: true,
			expectedID:  0,
			errorType:   ErrInvalidCredentials,
		},
		{
			name:     "multiple users with same username (should not happen but test first result)",
			username: "testuser",
			password: validPassword,
			setupMock: func(mock sqlmock.Sqlmock) {
				// QueryRow only returns the first row
				rows := sqlmock.NewRows([]string{"id", "password"}).
					AddRow(1, validHash)
				mock.ExpectQuery("SELECT id, password FROM users WHERE username = \\$1").
					WithArgs("testuser").
					WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET last_login = \\$1 WHERE id = \\$2").
					WithArgs(sqlmock.AnyArg(), 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectError: false,
			expectedID:  1,
		},
		{
			name:     "special characters in username",
			username: "test@user.com",
			password: validPassword,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "password"}).
					AddRow(5, validHash)
				mock.ExpectQuery("SELECT id, password FROM users WHERE username = \\$1").
					WithArgs("test@user.com").
					WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET last_login = \\$1 WHERE id = \\$2").
					WithArgs(sqlmock.AnyArg(), 5).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectError: false,
			expectedID:  5,
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

			model := NewUserModel(db)
			id, err := model.Authenticate(tt.username, tt.password)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
				assert.Equal(t, tt.expectedID, id)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, id)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserModelAuthenticateLastLoginUpdate(t *testing.T) {
	validPassword := "test123"
	validHash, err := bcrypt.GenerateFromPassword([]byte(validPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to generate test password hash: %v", err)
	}

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() {
		_ = db.Close()
	}()

	rows := sqlmock.NewRows([]string{"id", "password"}).
		AddRow(1, validHash)
	mock.ExpectQuery("SELECT id, password FROM users WHERE username = \\$1").
		WithArgs("testuser").
		WillReturnRows(rows)

	mock.ExpectExec("UPDATE users SET last_login = \\$1 WHERE id = \\$2").
		WithArgs(sqlmock.AnyArg(), 1).
		WillReturnResult(sqlmock.NewResult(0, 1)).
		WillDelayFor(0)

	model := NewUserModel(db)
	id, err := model.Authenticate("testuser", validPassword)

	assert.NoError(t, err)
	assert.Equal(t, 1, id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

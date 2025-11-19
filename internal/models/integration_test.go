//go:build integration

package models

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/lib/pq"
)

var db *sql.DB

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "16-alpine",
		Env: []string{
			"POSTGRES_PASSWORD=secret",
			"POSTGRES_USER=swimmate",
			"POSTGRES_DB=swimmate_test",
			"listen_addresses = '*'",
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	hostAndPort := resource.GetHostPort("5432/tcp")
	databaseUrl := fmt.Sprintf("postgres://swimmate:secret@%s/swimmate_test?sslmode=disable", hostAndPort)

	log.Println("Connecting to database on url: ", databaseUrl)

	resource.Expire(120)

	pool.MaxWait = 120 * time.Second
	if err = pool.Retry(func() error {
		db, err = sql.Open("postgres", databaseUrl)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// Create schema
	if err := createSchema(db); err != nil {
		log.Fatalf("Could not create schema: %s", err)
	}

	code := m.Run()

	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func createSchema(db *sql.DB) error {
	schema := `
		CREATE SEQUENCE IF NOT EXISTS auth_user_id_seq;
		CREATE SEQUENCE IF NOT EXISTS tracks_track_id_seq;

		CREATE TABLE IF NOT EXISTS users (
			id integer PRIMARY KEY DEFAULT nextval('auth_user_id_seq'),
			password character varying(128) NOT NULL,
			last_login timestamp with time zone,
			username character varying(150) NOT NULL UNIQUE,
			first_name character varying(150) NOT NULL,
			last_name character varying(150) NOT NULL,
			email character varying(254) NOT NULL,
			date_joined timestamp with time zone NOT NULL
		);

		CREATE TABLE IF NOT EXISTS swims (
			id bigint PRIMARY KEY DEFAULT nextval('tracks_track_id_seq'),
			date date NOT NULL,
			distance_m integer NOT NULL,
			assessment integer NOT NULL,
			user_id integer NOT NULL REFERENCES users(id)
		);

		CREATE INDEX IF NOT EXISTS idx_swims_user_id ON swims(user_id);
		CREATE INDEX IF NOT EXISTS idx_swims_date ON swims(date);
	`

	_, err := db.Exec(schema)
	return err
}

func cleanupTables(t *testing.T) {
	t.Helper()
	_, err := db.Exec("TRUNCATE swims, users RESTART IDENTITY CASCADE")
	assert.NoError(t, err)
}

func TestIntegrationUserAuthentication(t *testing.T) {
	cleanupTables(t)

	userModel := NewUserModel(db)

	// Create a test user
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	assert.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO users (username, password, first_name, last_name, email, date_joined)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, "testuser", string(hashedPassword), "Test", "User", "test@example.com", time.Now())
	assert.NoError(t, err)

	tests := []struct {
		name        string
		username    string
		password    string
		expectError bool
		errorType   error
	}{
		{
			name:        "successful authentication",
			username:    "testuser",
			password:    "password123",
			expectError: false,
		},
		{
			name:        "invalid username",
			username:    "nonexistent",
			password:    "password123",
			expectError: true,
			errorType:   ErrInvalidCredentials,
		},
		{
			name:        "invalid password",
			username:    "testuser",
			password:    "wrongpassword",
			expectError: true,
			errorType:   ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := userModel.Authenticate(tt.username, tt.password)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
				assert.Equal(t, 0, id)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, 0, id)

				// Verify last_login was updated
				var lastLogin time.Time
				err = db.QueryRow("SELECT last_login FROM users WHERE id = $1", id).Scan(&lastLogin)
				assert.NoError(t, err)
				assert.WithinDuration(t, time.Now(), lastLogin, 5*time.Second)
			}
		})
	}
}

func TestIntegrationSwimCRUD(t *testing.T) {
	cleanupTables(t)

	swimModel := NewSwimModel(db)

	// Create a test user
	var userID int
	err := db.QueryRow(`
		INSERT INTO users (username, password, first_name, last_name, email, date_joined)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, "testuser", "hashedpassword", "Test", "User", "test@example.com", time.Now()).Scan(&userID)
	assert.NoError(t, err)

	t.Run("insert and retrieve swim", func(t *testing.T) {
		date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		err := swimModel.Insert(date, 1500, 2, userID)
		assert.NoError(t, err)

		swim, err := swimModel.Get()
		assert.NoError(t, err)
		assert.Equal(t, date.Format("2006-01-02"), swim.Date.Format("2006-01-02"))
		assert.Equal(t, 1500, swim.DistanceM)
		assert.Equal(t, 2, swim.Assessment)
	})

	t.Run("get all swims ordered by date ASC", func(t *testing.T) {
		// Insert multiple swims
		dates := []time.Time{
			time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
		}

		for _, date := range dates {
			err := swimModel.Insert(date, 1000, 1, userID)
			assert.NoError(t, err)
		}

		swims, err := swimModel.GetAll(userID)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(swims), 3)

		// Verify ordering (ASC by date)
		for i := 1; i < len(swims); i++ {
			assert.True(t, swims[i].Date.After(swims[i-1].Date) || swims[i].Date.Equal(swims[i-1].Date))
		}
	})

	t.Run("pagination", func(t *testing.T) {
		// Get first page
		page1, err := swimModel.GetPaginated(userID, 2, 0, SwimSortDate, SortDirectionDesc)
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(page1), 2)

		// Get second page
		page2, err := swimModel.GetPaginated(userID, 2, 2, SwimSortDate, SortDirectionDesc)
		assert.NoError(t, err)

		// Verify DESC ordering (most recent first)
		if len(page1) >= 2 {
			assert.True(t, page1[0].Date.After(page1[1].Date) || page1[0].Date.Equal(page1[1].Date))
		}

		// Verify pages don't overlap
		if len(page1) > 0 && len(page2) > 0 {
			assert.NotEqual(t, page1[0].Date, page2[0].Date)
		}
	})
}

func TestIntegrationSummarize(t *testing.T) {
	cleanupTables(t)

	swimModel := NewSwimModel(db)

	// Create a test user
	var userID int
	err := db.QueryRow(`
		INSERT INTO users (username, password, first_name, last_name, email, date_joined)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, "testuser", "hashedpassword", "Test", "User", "test@example.com", time.Now()).Scan(&userID)
	assert.NoError(t, err)

	// Insert swims across different months and years
	testData := []struct {
		date       time.Time
		distanceM  int
		assessment int
	}{
		{time.Date(2023, 12, 15, 0, 0, 0, 0, time.UTC), 1000, 1},
		{time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), 1500, 2},
		{time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC), 2000, 2},
		{time.Date(2024, 2, 5, 0, 0, 0, 0, time.UTC), 1200, 1},
		{time.Now().AddDate(0, 0, -1), 800, 1}, // Yesterday (for weekly test)
	}

	for _, td := range testData {
		err := swimModel.Insert(td.date, td.distanceM, td.assessment, userID)
		assert.NoError(t, err)
	}

	summary := swimModel.Summarize(userID)

	t.Run("total aggregations", func(t *testing.T) {
		expectedTotal := 1000 + 1500 + 2000 + 1200 + 800
		assert.Equal(t, expectedTotal, summary.TotalDistance)
		assert.Equal(t, 5, summary.TotalCount)
	})

	t.Run("yearly aggregations", func(t *testing.T) {
		// 2023
		year2023, exists := summary.YearMap[2023]
		assert.True(t, exists)
		assert.Equal(t, 1, year2023.Count)
		assert.Equal(t, 1000, year2023.DistanceM)

		// 2024
		year2024, exists := summary.YearMap[2024]
		assert.True(t, exists)
		assert.Equal(t, 3, year2024.Count)
		assert.Equal(t, 4700, year2024.DistanceM)

		// Current year (yesterday's swim)
		currentYear := time.Now().Year()
		yearCurrent, exists := summary.YearMap[currentYear]
		assert.True(t, exists)
		assert.Equal(t, 1, yearCurrent.Count)
		assert.Equal(t, 800, yearCurrent.DistanceM)
	})

	t.Run("monthly aggregations", func(t *testing.T) {
		year2024 := summary.YearMap[2024]

		// All 12 months should be initialized
		assert.Equal(t, 12, len(year2024.MonthMap))

		// January 2024
		jan := year2024.MonthMap[time.January]
		assert.Equal(t, 2, jan.Count)
		assert.Equal(t, 3500, jan.DistanceM)

		// February 2024
		feb := year2024.MonthMap[time.February]
		assert.Equal(t, 1, feb.Count)
		assert.Equal(t, 1200, feb.DistanceM)

		// March 2024 (no swims)
		mar := year2024.MonthMap[time.March]
		assert.Equal(t, 0, mar.Count)
		assert.Equal(t, 0, mar.DistanceM)
	})

	t.Run("weekly aggregations", func(t *testing.T) {
		// We inserted a swim yesterday, which should be in the current week
		assert.GreaterOrEqual(t, summary.WeeklyCount, 1)
		assert.GreaterOrEqual(t, summary.WeeklyDistance, 800)
	})
}

func TestIntegrationMultipleUsers(t *testing.T) {
	cleanupTables(t)

	swimModel := NewSwimModel(db)

	// Create two users
	var user1ID, user2ID int
	err := db.QueryRow(`
		INSERT INTO users (username, password, first_name, last_name, email, date_joined)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, "user1", "pass1", "User", "One", "user1@example.com", time.Now()).Scan(&user1ID)
	assert.NoError(t, err)

	err = db.QueryRow(`
		INSERT INTO users (username, password, first_name, last_name, email, date_joined)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, "user2", "pass2", "User", "Two", "user2@example.com", time.Now()).Scan(&user2ID)
	assert.NoError(t, err)

	// Insert swims for both users
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	err = swimModel.Insert(date, 1000, 1, user1ID)
	assert.NoError(t, err)

	err = swimModel.Insert(date, 2000, 2, user2ID)
	assert.NoError(t, err)

	// Verify user isolation
	user1Swims, err := swimModel.GetAll(user1ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(user1Swims))
	assert.Equal(t, 1000, user1Swims[0].DistanceM)

	user2Swims, err := swimModel.GetAll(user2ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(user2Swims))
	assert.Equal(t, 2000, user2Swims[0].DistanceM)

	// Verify summaries are isolated
	summary1 := swimModel.Summarize(user1ID)
	assert.Equal(t, 1000, summary1.TotalDistance)
	assert.Equal(t, 1, summary1.TotalCount)

	summary2 := swimModel.Summarize(user2ID)
	assert.Equal(t, 2000, summary2.TotalDistance)
	assert.Equal(t, 1, summary2.TotalCount)
}

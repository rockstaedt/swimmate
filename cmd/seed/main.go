package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"

	_ "github.com/lib/pq"
)

func main() {
	// Command line flags
	username := flag.String("username", "", "Username for the new user (required)")
	password := flag.String("password", "password123", "Password for the new user (default: password123)")
	firstName := flag.String("first-name", "Test", "First name for the new user")
	lastName := flag.String("last-name", "User", "Last name for the new user")
	email := flag.String("email", "", "Email for the new user (auto-generated if not provided)")
	numSwims := flag.Int("swims", 50, "Number of swim entries to create")
	daysBack := flag.Int("days-back", 365, "Generate swims going back this many days")
	flag.Parse()

	// Validate required flags
	if *username == "" {
		log.Fatal("Error: -username flag is required")
	}

	// Auto-generate email if not provided
	if *email == "" {
		*email = fmt.Sprintf("%s@example.com", *username)
	}

	// Get database connection string from environment
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("Error: DB_DSN environment variable is not set")
	}

	// Connect to database
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	log.Println("Connected to database successfully")

	// Create user
	userID, err := createUser(db, *username, *password, *firstName, *lastName, *email)
	if err != nil {
		log.Fatalf("Error creating user: %v", err)
	}

	log.Printf("Created user '%s' (ID: %d) with password '%s'", *username, userID, *password)

	// Create swim entries
	err = createSwims(db, userID, *numSwims, *daysBack)
	if err != nil {
		log.Fatalf("Error creating swims: %v", err)
	}

	log.Printf("Successfully created %d swim entries for user '%s'", *numSwims, *username)
	log.Println("Seeding completed!")
}

func createUser(db *sql.DB, username, password, firstName, lastName, email string) (int, error) {
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password: %w", err)
	}

	// Insert user into database
	stmt := `
		INSERT INTO users (username, password, first_name, last_name, email, date_joined, last_login)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	var userID int
	now := time.Now()
	err = db.QueryRow(stmt, username, hashedPassword, firstName, lastName, email, now, now).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("failed to insert user: %w", err)
	}

	return userID, nil
}

func createSwims(db *sql.DB, userID, numSwims, daysBack int) error {
	// Create a local random number generator
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	stmt := `INSERT INTO swims (date, distance_m, assessment, user_id) VALUES ($1, $2, $3, $4)`

	// Generate random swims
	for i := 0; i < numSwims; i++ {
		// Random date within the specified range
		daysAgo := rng.Intn(daysBack)
		swimDate := time.Now().AddDate(0, 0, -daysAgo)

		// Random distance (realistic swimming distances in meters)
		// Typical distances: 500m, 1000m, 1500m, 2000m, 2500m, 3000m, etc.
		distanceOptions := []int{500, 750, 1000, 1200, 1500, 1800, 2000, 2500, 3000, 3500, 4000}
		distance := distanceOptions[rng.Intn(len(distanceOptions))]

		// Random assessment (0-2 stars)
		assessment := rng.Intn(3)

		_, err := db.Exec(stmt, swimDate, distance, assessment, userID)
		if err != nil {
			return fmt.Errorf("failed to insert swim entry: %w", err)
		}
	}

	return nil
}

package config

import (
	"log"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// ExecuteMigrations runs all pending database migrations
func ExecuteMigrations() {
	// Get database connection
	db, err := DB.DB()
	if err != nil {
		log.Fatal("Failed to get database connection:", err)
	}

	// Create postgres driver
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatal("Failed to create postgres driver:", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithDatabaseInstance(
		"file://"+filepath.Join("migrations"),
		"postgres",
		driver,
	)
	if err != nil {
		log.Fatal("Failed to create migrate instance:", err)
	}

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal("Failed to run migrations:", err)
	}

	log.Println("Database migrations completed successfully")
}

// RollbackMigration rolls back the last migration
func RollbackMigration() {
	// Get database connection
	db, err := DB.DB()
	if err != nil {
		log.Fatal("Failed to get database connection:", err)
	}

	// Create postgres driver
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatal("Failed to create postgres driver:", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithDatabaseInstance(
		"file://"+filepath.Join("migrations"),
		"postgres",
		driver,
	)
	if err != nil {
		log.Fatal("Failed to create migrate instance:", err)
	}

	// Rollback migration
	if err := m.Steps(-1); err != nil {
		log.Fatal("Failed to rollback migration:", err)
	}

	log.Println("Migration rolled back successfully")
} 
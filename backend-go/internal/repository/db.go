// Package repository provides PostgreSQL data access layer.
package repository

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"

	"github.com/aspira/aspira-pay/internal/config"
)

// DB wraps sql.DB with convenience methods.
type DB struct {
	*sql.DB
}

// New connects to PostgreSQL and returns a DB instance.
func New(cfg config.DatabaseConfig) (*DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("repository: cannot open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MaxConns / 2)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("repository: cannot ping database: %w", err)
	}

	return &DB{DB: db}, nil
}

// RunMigrations executes all migration files in order.
func (db *DB) RunMigrations(migrationsPath string) error {
	// In Sandbox mode, migrations are run via init_db.sh or manually.
	// This method is a placeholder for future auto-migration support.
	return nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.DB.Close()
}

// BeginTx starts a new transaction.
func (db *DB) BeginTx() (*sql.Tx, error) {
	return db.DB.Begin()
}

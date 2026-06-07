// Package repository provides PostgreSQL data access layer.
package repository

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"github.com/aspira/aspira-pay/internal/config"
)

// DB wraps sql.DB with convenience methods.
type DB struct {
	*sql.DB
}

// New connects to PostgreSQL and returns a DB instance.
// Connection pool tuned for production workloads:
//   - max_conns: 25 per instance (design doc §13.1: 200 total across all services)
//   - max_idle: 10 (keep warm connections ready, reduce connect latency)
//   - conn_max_lifetime: 30min (prevent stale connections after network blips)
//
// For multi-replica deployments, use PgBouncer (transaction mode) in front
// of PostgreSQL to pool connections from all service instances.
func New(cfg config.DatabaseConfig) (*DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("repository: cannot open database: %w", err)
	}

	// Connection pool sizing
	if cfg.MaxConns == 0 {
		cfg.MaxConns = 25
	}
	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MaxConns / 2)
	db.SetConnMaxLifetime(30 * time.Minute) // Recycle connections periodically
	db.SetConnMaxIdleTime(5 * time.Minute)   // Close idle connections to free PG resources

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

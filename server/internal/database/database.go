// Package database provides the Ent client and database connection helpers.
package database

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/nemvince/fog-next/ent"
	"github.com/nemvince/fog-next/internal/config"
)

// Open creates and verifies an Ent client connected to PostgreSQL.
func Open(ctx context.Context, cfg config.DatabaseConfig) (*ent.Client, error) {
	drv, err := sql.Open(dialect.Postgres, cfg.DSNString())
	if err != nil {
		return nil, fmt.Errorf("opening postgres driver: %w", err)
	}

	db := drv.DB()
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := db.PingContext(ctx); err != nil {
		drv.Close()
		return nil, fmt.Errorf("pinging postgres: %w", err)
	}

	return ent.NewClient(ent.Driver(drv)), nil
}

// Migrate runs automatic schema migration using Ent's built-in migrator.
// In production you should use versioned Atlas migrations instead.
func Migrate(ctx context.Context, client *ent.Client) error {
	if err := client.Schema.Create(ctx); err != nil {
		return fmt.Errorf("running ent schema migration: %w", err)
	}
	return nil
}


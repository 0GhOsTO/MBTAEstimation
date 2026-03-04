package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

func initDB() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		host := getEnvOrDefault("DB_HOST", "localhost")
		port := getEnvOrDefault("DB_PORT", "5432")
		user := getEnvOrDefault("DB_USER", "postgres")
		password := os.Getenv("DB_PASSWORD")
		dbName := getEnvOrDefault("DB_NAME", "timeseries")
		sslmode := getEnvOrDefault("DB_SSLMODE", "require")
		databaseURL = buildPostgresURL(host, port, user, password, dbName, sslmode)
	}

	var err error
	db, err = sql.Open("postgres", databaseURL)
	if err != nil {
		log.Printf("⚠️ failed to open database connection: %v", err)
		return
	}

	db.SetMaxOpenConns(30)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Printf("⚠️ failed to ping database: %v", err)
		db = nil
		return
	}

	if err := initializeSchema(context.Background()); err != nil {
		log.Printf("⚠️ failed to initialize schema: %v", err)
	}

	log.Println("✅ database initialized")
}

func buildPostgresURL(host, port, user, password, dbName, sslmode string) string {
	uri := &url.URL{
		Scheme: "postgres",
		Host:   fmt.Sprintf("%s:%s", host, port),
		Path:   dbName,
		User:   url.UserPassword(user, password),
	}

	query := uri.Query()
	query.Set("sslmode", sslmode)
	uri.RawQuery = query.Encode()

	return uri.String()
}

func getEnvOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func initializeSchema(ctx context.Context) error {
	statements := []string{
		`CREATE EXTENSION IF NOT EXISTS timescaledb`,
		`CREATE TABLE IF NOT EXISTS prediction_snapshots (
			id BIGSERIAL PRIMARY KEY,
			observed_at TIMESTAMPTZ NOT NULL,
			station_id TEXT NOT NULL,
			direction SMALLINT NOT NULL,
			vehicle_id TEXT NOT NULL,
			trip_id TEXT,
			predicted_arrival TIMESTAMPTZ,
			status TEXT,
			graded BOOLEAN NOT NULL DEFAULT FALSE,
			actual_arrival TIMESTAMPTZ,
			error_seconds DOUBLE PRECISION,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (observed_at, station_id, direction, vehicle_id, predicted_arrival)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_prediction_lookup
			ON prediction_snapshots (station_id, direction, vehicle_id, observed_at DESC)
			WHERE graded = FALSE`,
		`CREATE INDEX IF NOT EXISTS idx_prediction_station_agg
			ON prediction_snapshots (station_id, direction)
			WHERE graded = TRUE`,
	}

	for _, stmt := range statements {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			if strings.Contains(strings.ToLower(stmt), "timescaledb") {
				log.Printf("⚠️ timescaledb extension unavailable, continuing with standard PostgreSQL: %v", err)
				continue
			}
			return err
		}
	}

	_, _ = db.ExecContext(ctx, `SELECT create_hypertable('prediction_snapshots', 'observed_at', if_not_exists => TRUE)`)
	return nil
}

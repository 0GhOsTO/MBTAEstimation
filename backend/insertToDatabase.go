package main

import (
	"context"
	"time"

	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

var db *sql.DB

func initDB() {
	var err error
	// replace host, port, password as needed
	connStr := "postgres://postgres:postgres@localhost:5432/timeseries?sslmode=disable"
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Cannot ping database: %v", err)
	}
}

func InsertPredictionToDatabase(
	tripID string,
	predictedArrival time.Time,
	actualArrival time.Time,
) error {

	lossSeconds := actualArrival.Sub(predictedArrival).Seconds()

	query := `
		INSERT INTO arrival_predictions_pktrm (
			trip_id,
			predicted_arrival,
			actual_arrival,
			loss_seconds
		)
		VALUES ($1, $2, $3, $4)
	`

	_, err := db.ExecContext(
		context.Background(),
		query,
		tripID,
		predictedArrival,
		actualArrival,
		lossSeconds,
	)

	if err != nil {
		log.Printf("Failed to insert prediction: %v", err)
		return err
	}

	return nil
}

func main_test() {
	initDB()

	tripID := "trip_123"

	predicted := time.Date(2026, 2, 5, 15, 30, 0, 0, time.UTC)
	actual := time.Date(2026, 2, 5, 15, 32, 15, 0, time.UTC)

	err := InsertPredictionToDatabase(tripID, predicted, actual)
	if err != nil {
		log.Fatalf("Insert failed: %v", err)
	}

	log.Println("Prediction inserted successfully!")
}

// Requirement before running: create go module
// cd f:\CS\ProjectGithub\MBTAEstimation\backend; go mod init mbta-backend
// Create .env file in backend folder with content:
// MBTA_API_KEY=your_actual_api_key_here
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

// grab the API key.
var key string

// stop id: {in/outbound : {vehicle id: prediction data}}}
var predictionDataMap = make(map[string]map[int]map[string][]PredictionData)

// mutex for protecting prediction map from concurrent access
var predictionMutex sync.RWMutex

// Prediction accuracy statistics
var totalPredictions int
var correctPredictions int
var incorrectPredictions int
var statsMutex sync.Mutex

// Struct to hold prediction data
// 1. observation time
// 2. stop ID
// 3. PREDICTED arrival time
// 4. PREDICTED departure time
// 5. current status
type PredictionData struct {
	ObservationTime time.Time
	StopID          string
	ArrivalTime     *time.Time // pointer because it can be null
	DepartureTime   *time.Time // pointer because it can be null
	Status          string
	VehicleID       string
	TripID          string
}

// Struct to unmarshal MBTA predictions API response
type PredictionsResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Attributes struct {
			ArrivalTime   *string `json:"arrival_time"`   // ISO 8601 timestamp or null
			DepartureTime *string `json:"departure_time"` // ISO 8601 timestamp or null
			Status        *string `json:"status"`         // can be null
		} `json:"attributes"`
		Relationships struct {
			Stop struct {
				Data struct {
					ID   string `json:"id"`
					Type string `json:"type"`
				} `json:"data"`
			} `json:"stop"`
			Vehicle struct {
				Data struct {
					ID   string `json:"id"`
					Type string `json:"type"`
				} `json:"data"`
			} `json:"vehicle"`
			Trip struct {
				Data struct {
					ID   string `json:"id"`
					Type string `json:"type"`
				} `json:"data"`
			} `json:"trip"`
		} `json:"relationships"`
	} `json:"data"`
}

func init() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("No .env found")
	}
	key = os.Getenv("MBTA_API_KEY")
	if key == "" {
		panic("MBTA_API_KEY not set")
	}
}

func fetchPrediction_single(stopID string, direction int) (PredictionData, string, error) {
	// constructing the request.
	//url := fmt.Sprintf("https://api-v3.mbta.com/predictions?filter[stop]=%s&filter[direction_id]=%d", stopID, direction)
	// gets the next two predictions for the stop and direction, sorted by arrival time
	url := fmt.Sprintf("https://api-v3.mbta.com/predictions?filter[stop]=%s&filter[direction_id]=%d&sort=arrival_time&page[limit]=2", stopID, direction)
	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return PredictionData{}, "nil", err
	}
	req.Header.Set("x-api-key", key)
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return PredictionData{}, "nil", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return PredictionData{}, "nil", err
	}

	// Unmarshal JSON response
	var predResp PredictionsResponse
	if err := json.Unmarshal(body, &predResp); err != nil {
		return PredictionData{}, "nil", fmt.Errorf("failed to unmarshal predictions: %w", err)
	}

	// Extract the fields you need
	predictions := make([]PredictionData, 0, len(predResp.Data))
	observationTime := time.Now() // Current time as observation time

	for _, pred := range predResp.Data {
		data := PredictionData{
			ObservationTime: observationTime,
			StopID:          pred.Relationships.Stop.Data.ID,
			VehicleID:       pred.Relationships.Vehicle.Data.ID,
			TripID:          pred.Relationships.Trip.Data.ID,
		}

		// Parse arrival time if present
		if pred.Attributes.ArrivalTime != nil && *pred.Attributes.ArrivalTime != "" {
			if arrivalTime, err := time.Parse(time.RFC3339, *pred.Attributes.ArrivalTime); err == nil {
				data.ArrivalTime = &arrivalTime
			}
		}

		// Parse departure time if present
		if pred.Attributes.DepartureTime != nil && *pred.Attributes.DepartureTime != "" {
			if departureTime, err := time.Parse(time.RFC3339, *pred.Attributes.DepartureTime); err == nil {
				data.DepartureTime = &departureTime
			}
		}

		// Set status if present
		if pred.Attributes.Status != nil {
			data.Status = *pred.Attributes.Status
		}

		predictions = append(predictions, data)
	}

	// Return the second prediction
	res := predictions[1]
	// Extract vehicle ID of the second prediction
	vehicle := res.VehicleID

	// Store the prediction data in the map.
	predictionMutex.Lock()
	if predictionDataMap[stopID] == nil {
		predictionDataMap[stopID] = make(map[int]map[string][]PredictionData)
	}
	if predictionDataMap[stopID][direction] == nil {
		predictionDataMap[stopID][direction] = make(map[string][]PredictionData)
	}
	predictionDataMap[stopID][direction][vehicle] = append(predictionDataMap[stopID][direction][vehicle], res)
	predictionMutex.Unlock()

	fmt.Printf("Stored prediction: Stop %s, Dir %d, Vehicle %s (Arrival: %v)\n", stopID, direction, vehicle, res.ArrivalTime)

	return res, vehicle, nil
}

func main_test_pred() {
	// 1. Every 30 seconds, fetch prediction for a specific stop ID.
	// 2. Store the fetched data until the vehicle arrives.
	// 3. Once the vehicle arrived, grade the prediction accuracy.

	// 1. Every 30 seconds ...
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	// call once right away
	for {
		<-ticker.C
		// @TODO: NEED to handle in go routine
		// NEED to handle if there is no data returned(happens sometimes due to bug in MBTA API)
		// 1. Require to request the prediction
		// 2.
		// go function call
		// Uncertainty by station vs uncertainty by the train ID.

		//=======TESTING==========
		pred, _, err := fetchPrediction_single("70135", 0) // example stop ID
		if err != nil {
			panic(err)
		}

		// Print extracted data
		fmt.Println("=== Prediction ===")
		fmt.Printf("Observation Time: %s\n", pred.ObservationTime.Format(time.RFC3339))
		fmt.Printf("Stop ID: %s\n", pred.StopID)
		fmt.Printf("Vehicle ID: %s\n", pred.VehicleID)
		fmt.Printf("Trip ID: %s\n", pred.TripID)
		if pred.ArrivalTime != nil {
			fmt.Printf("Predicted Arrival: %s\n", pred.ArrivalTime.Format(time.RFC3339))
		} else {
			fmt.Println("Predicted Arrival: N/A")
		}
		if pred.DepartureTime != nil {
			fmt.Printf("Predicted Departure: %s\n", pred.DepartureTime.Format(time.RFC3339))
		} else {
			fmt.Println("Predicted Departure: N/A")
		}
		fmt.Printf("Status: %s\n", pred.Status)
		fmt.Println()
	}
}

func main() {
	// Start goroutine to constantly listen for arrivals
	go func() {
		fmt.Println("Started listening to ArrivalChannel...")
		// this is equivalent to <-Arrival Channel.
		for arrival := range ArrivalChannel {
			fmt.Printf("\n=== TRAIN ARRIVAL DETECTED ===\n")
			fmt.Printf("Station (Place ID): %s\n", arrival.StationPlaceID)
			fmt.Printf("Station (Stop ID): %s\n", arrival.StationStopID)
			fmt.Printf("Train ID: %s\n", arrival.TrainID)
			fmt.Printf("Direction: %d\n", arrival.Direction)
			fmt.Printf("Actual Arrival Time: %s\n", arrival.ArrivalTime.Format(time.RFC3339))
			fmt.Printf("==============================\n\n")

			// Compare with prediction data and score accuracy
			predictionMutex.RLock()
			if predictions, exists := predictionDataMap[arrival.StationStopID][arrival.Direction][arrival.TrainID]; exists {
				fmt.Printf("Found %d prediction(s) for this train:\n", len(predictions))

				for i, pred := range predictions {
					if pred.ArrivalTime != nil {
						// Calculate difference between predicted and actual
						difference := arrival.ArrivalTime.Sub(*pred.ArrivalTime)
						diffSeconds := difference.Seconds()
						diffMinutes := difference.Minutes()

						fmt.Printf("\nPrediction #%d:\n", i+1)
						fmt.Printf("  Predicted Arrival: %s\n", pred.ArrivalTime.Format(time.RFC3339))
						fmt.Printf("  Actual Arrival:    %s\n", arrival.ArrivalTime.Format(time.RFC3339))
						fmt.Printf("  Difference:        %.0f seconds (%.2f minutes)\n", diffSeconds, diffMinutes)

						// Score the prediction (smaller difference = better score)
						absMinutes := math.Abs(diffMinutes)
						var score string
						var isCorrect bool

						// Define "correct" as within 3 minutes of actual arrival
						if absMinutes <= 3 {
							isCorrect = true
							if absMinutes <= 1 {
								score = "EXCELLENT (CORRECT)"
							} else {
								score = "GOOD (CORRECT)"
							}
						} else {
							isCorrect = false
							if absMinutes <= 5 {
								score = "FAIR (WRONG)"
							} else {
								score = "POOR (WRONG)"
							}
						}

						if diffSeconds > 0 {
							fmt.Printf("  Status:            Train arrived %.2f minutes LATE\n", diffMinutes)
						} else if diffSeconds < 0 {
							fmt.Printf("  Status:            Train arrived %.2f minutes EARLY\n", math.Abs(diffMinutes))
						} else {
							fmt.Printf("  Status:            Train arrived EXACTLY on time!\n")
						}
						fmt.Printf("  Accuracy Score:    %s\n", score)

						// Update statistics
						statsMutex.Lock()
						totalPredictions++
						if isCorrect {
							correctPredictions++
						} else {
							incorrectPredictions++
						}
						accuracy := float64(correctPredictions) / float64(totalPredictions) * 100
						fmt.Printf("\n📊 OVERALL STATISTICS:\n")
						fmt.Printf("   Total Predictions: %d\n", totalPredictions)
						fmt.Printf("   Correct (≤3 min): %d\n", correctPredictions)
						fmt.Printf("   Wrong (>3 min):   %d\n", incorrectPredictions)
						fmt.Printf("   Accuracy Rate:    %.2f%%\n", accuracy)
						statsMutex.Unlock()
					} else {
						fmt.Printf("\nPrediction #%d: No predicted arrival time available\n", i+1)
					}
				}
			} else {
				fmt.Println("No predictions found for this train arrival")
			}
			predictionMutex.RUnlock()

			fmt.Println("\n==============================\n")
		}
	}()

	// 1. Every 3 minutes...fetch the prediction
	ticker := time.NewTicker(180 * time.Second)
	defer ticker.Stop()
	for {
		<-ticker.C
		for _, id := range parentStationIDs {
			// fetch the prediction for both directions in parallel.
			go func(stopID string) {
				go fetchPrediction_single(stopID, 0)
				go fetchPrediction_single(stopID, 1)
			}(id)
		}
	}
}

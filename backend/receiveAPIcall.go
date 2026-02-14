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

// StationStats holds accuracy statistics for a station and direction
type StationStats struct {
	Total     int
	Correct   int
	Incorrect int
}

// Prediction accuracy statistics by station and direction
// Map structure: stationID -> direction (0=outbound, 1=inbound) -> stats
var stationAccuracyMap = make(map[string]map[int]*StationStats)
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

	// Check if we have at least 2 predictions
	if len(predictions) < 2 {
		return PredictionData{}, "nil", fmt.Errorf("need at least 2 predictions, got %d", len(predictions))
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

// cleanupOldPredictions removes predictions older than 30 minutes to prevent memory leak
func cleanupOldPredictions() {
	predictionMutex.Lock()
	defer predictionMutex.Unlock()

	cutoffTime := time.Now().Add(-30 * time.Minute)
	cleanedCount := 0

	for stopID, directions := range predictionDataMap {
		for direction, vehicles := range directions {
			for vehicleID, predictions := range vehicles {
				// Filter out old predictions
				validPredictions := make([]PredictionData, 0)
				for _, pred := range predictions {
					if pred.ObservationTime.After(cutoffTime) {
						validPredictions = append(validPredictions, pred)
					} else {
						cleanedCount++
					}
				}

				if len(validPredictions) > 0 {
					predictionDataMap[stopID][direction][vehicleID] = validPredictions
				} else {
					delete(predictionDataMap[stopID][direction], vehicleID)
				}
			}
			// Clean up empty direction maps
			if len(predictionDataMap[stopID][direction]) == 0 {
				delete(predictionDataMap[stopID], direction)
			}
		}
		// Clean up empty stop maps
		if len(predictionDataMap[stopID]) == 0 {
			delete(predictionDataMap, stopID)
		}
	}

	if cleanedCount > 0 {
		fmt.Printf("🧹 Cleaned up %d old predictions (older than 30 minutes)\n", cleanedCount)
	}
}

// StationAccuracyResponse represents the JSON response for station accuracy data
type StationAccuracyResponse struct {
	StationID        string  `json:"station_id"`
	StationName      string  `json:"station_name"`
	InboundAccuracy  float64 `json:"inbound_accuracy"`
	InboundTotal     int     `json:"inbound_total"`
	OutboundAccuracy float64 `json:"outbound_accuracy"`
	OutboundTotal    int     `json:"outbound_total"`
}

// HTTP handler to get all station statistics
func handleGetStatistics(w http.ResponseWriter, r *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	statsMutex.Lock()
	defer statsMutex.Unlock()

	// Map of station IDs to friendly names
	stationNames := map[string]string{
		"place-lake":  "Boston College",
		"place-sougr": "South Street",
		"place-chill": "Chestnut Hill Avenue",
		"place-chswk": "Chiswick Road",
		"place-sthld": "Sutherland Road",
		"place-wascm": "Washington Street",
		"place-wrnst": "Warren Street",
		"place-alsgr": "Allston Street",
		"place-grigg": "Griggs Street",
		"place-harvd": "Harvard Avenue",
		"place-brico": "Packards Corner",
		"place-bucen": "Boston University Central",
		"place-buest": "Boston University East",
		"place-bland": "Blandford Street",
		"place-kencl": "Kenmore",
		"place-hymnl": "Hynes Convention Center",
		"place-coecl": "Copley",
		"place-armnl": "Arlington",
		"place-boyls": "Boylston",
		"place-pktrm": "Park Street",
	}

	response := make([]StationAccuracyResponse, 0)

	for stationID, directions := range stationAccuracyMap {
		stationResp := StationAccuracyResponse{
			StationID:   stationID,
			StationName: stationNames[stationID],
		}
		if stationResp.StationName == "" {
			stationResp.StationName = stationID
		}

		// Get inbound stats (direction 1)
		if stats, exists := directions[1]; exists && stats.Total > 0 {
			stationResp.InboundAccuracy = float64(stats.Correct) / float64(stats.Total) * 100
			stationResp.InboundTotal = stats.Total
		}

		// Get outbound stats (direction 0)
		if stats, exists := directions[0]; exists && stats.Total > 0 {
			stationResp.OutboundAccuracy = float64(stats.Correct) / float64(stats.Total) * 100
			stationResp.OutboundTotal = stats.Total
		}

		response = append(response, stationResp)
	}

	json.NewEncoder(w).Encode(response)
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
	// Start HTTP API server for frontend
	go func() {
		http.HandleFunc("/api/statistics", handleGetStatistics)
		fmt.Println("🌐 HTTP API server starting on http://localhost:8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			fmt.Printf("❌ HTTP server error: %v\n", err)
		}
	}()

	// Start periodic cleanup of old predictions (every 5 minutes)
	go func() {
		cleanupTicker := time.NewTicker(30 * time.Minute)
		defer cleanupTicker.Stop()
		fmt.Println("Started periodic prediction cleanup (every 30 minutes)...")
		for {
			<-cleanupTicker.C
			cleanupOldPredictions()
		}
	}()

	// Start goroutine to monitor actual train arrivals
	go actualArrivalMoment("Green-B")
	fmt.Println("Started monitoring Green-B line for arrivals...")
	// =============================LISTENING====================================
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
			// Use StationPlaceID (parent station) to match how predictions are stored
			predictionMutex.RLock()
			fmt.Printf("Looking up predictions for Place ID: %s, Direction: %d, Train: %s\n", arrival.StationPlaceID, arrival.Direction, arrival.TrainID)
			if predictions, exists := predictionDataMap[arrival.StationPlaceID][arrival.Direction][arrival.TrainID]; exists {
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
						if absMinutes <= 5 {
							isCorrect = true
							if absMinutes <= 3 {
								score = "EXCELLENT (CORRECT)"
							} else {
								score = "GOOD (CORRECT)"
							}
						} else {
							isCorrect = false
							if absMinutes <= 7 {
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

						// Update statistics per station and direction
						statsMutex.Lock()
						// Initialize station map if needed
						if stationAccuracyMap[arrival.StationPlaceID] == nil {
							stationAccuracyMap[arrival.StationPlaceID] = make(map[int]*StationStats)
						}
						// Initialize direction stats if needed
						if stationAccuracyMap[arrival.StationPlaceID][arrival.Direction] == nil {
							stationAccuracyMap[arrival.StationPlaceID][arrival.Direction] = &StationStats{}
						}

						stats := stationAccuracyMap[arrival.StationPlaceID][arrival.Direction]
						stats.Total++
						if isCorrect {
							stats.Correct++
						} else {
							stats.Incorrect++
						}

						accuracy := float64(stats.Correct) / float64(stats.Total) * 100
						directionName := "Outbound"
						if arrival.Direction == 1 {
							directionName = "Inbound"
						}

						fmt.Printf("\n📊 STATION STATISTICS [%s - %s]:\n", arrival.StationPlaceID, directionName)
						fmt.Printf("   Total Predictions: %d\n", stats.Total)
						fmt.Printf("   Correct (≤5 min): %d\n", stats.Correct)
						fmt.Printf("   Wrong (>5 min):   %d\n", stats.Incorrect)
						fmt.Printf("   Accuracy Rate:    %.2f%%\n", accuracy)
						statsMutex.Unlock()
					} else {
						fmt.Printf("\nPrediction #%d: No predicted arrival time available\n", i+1)
					}
				}

				// Cleanup: Remove evaluated predictions to prevent memory leak
				predictionMutex.RUnlock()
				predictionMutex.Lock()
				delete(predictionDataMap[arrival.StationPlaceID][arrival.Direction], arrival.TrainID)
				if len(predictionDataMap[arrival.StationPlaceID][arrival.Direction]) == 0 {
					delete(predictionDataMap[arrival.StationPlaceID], arrival.Direction)
				}
				if len(predictionDataMap[arrival.StationPlaceID]) == 0 {
					delete(predictionDataMap, arrival.StationPlaceID)
				}
				predictionMutex.Unlock()
				fmt.Printf("Cleaned up predictions for vehicle %s at station %s\n", arrival.TrainID, arrival.StationPlaceID)
			} else {
				fmt.Println("No predictions found for this train arrival")
				predictionMutex.RUnlock()
			}

			fmt.Println("\n==============================\n")
		}
	}()

	// Start goroutine to fetch predictions every 3 minutes
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		fmt.Println("Started periodic prediction fetching (every 3 minutes)...")

		// Fetch predictions immediately on startup
		fmt.Println("Fetching initial predictions...")
		for _, id := range parentStationIDs {
			go func(stopID string) {
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf("⚠️  Recovered from panic in fetchPrediction_single for stop %s: %v\n", stopID, r)
					}
				}()

				if _, _, err := fetchPrediction_single(stopID, 0); err != nil {
					fmt.Printf("⚠️  Error fetching prediction for stop %s direction 0: %v\n", stopID, err)
				}
				if _, _, err := fetchPrediction_single(stopID, 1); err != nil {
					fmt.Printf("⚠️  Error fetching prediction for stop %s direction 1: %v\n", stopID, err)
				}
			}(id)
		}

		// Then continue with periodic updates
		for {
			<-ticker.C
			fmt.Println("Fetching updated predictions...")
			for _, id := range parentStationIDs {
				// fetch the prediction for both directions in parallel with error handling
				go func(stopID string) {
					defer func() {
						if r := recover(); r != nil {
							fmt.Printf("⚠️  Recovered from panic in fetchPrediction_single for stop %s: %v\n", stopID, r)
						}
					}()

					if _, _, err := fetchPrediction_single(stopID, 0); err != nil {
						fmt.Printf("⚠️  Error fetching prediction for stop %s direction 0: %v\n", stopID, err)
					}
					if _, _, err := fetchPrediction_single(stopID, 1); err != nil {
						fmt.Printf("⚠️  Error fetching prediction for stop %s direction 1: %v\n", stopID, err)
					}
				}(id)
			}
		}
	}()

	// Block forever to keep the program running while goroutines work
	select {}
}

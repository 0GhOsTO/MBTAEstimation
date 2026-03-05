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

// Cached accuracy rates for quick API responses
// Map structure: stationID -> StationAccuracyResponse (with calculated accuracy)
var cachedAccuracyMap = make(map[string]*StationAccuracyResponse)

// Map of all Green Line B station IDs to friendly names
var greenBStationNames = map[string]string{
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
	// Orphan platforms (no parent station in MBTA API)
	"70136":       "Babcock Street",
	"70140":       "Amory Street",
	"place-bucen": "Boston University Central",
	"place-buest": "Boston University East",
	"place-bland": "Blandford Street",
	"place-kencl": "Kenmore",
	"place-hymnl": "Hynes Convention Center",
	"place-coecl": "Copley",
	"place-armnl": "Arlington",
	"place-boyls": "Boylston",
	"place-pktrm": "Park Street",
	"place-gover": "Government Center",
}

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
		fmt.Println("No .env file found (this is normal in production)")
	}
	key = os.Getenv("MBTA_API_KEY")
	if key == "" {
		panic("MBTA_API_KEY environment variable not set! Please set it in your deployment platform's environment variables.")
	}
	fmt.Println("MBTA API key loaded successfully")
}

func fetchPrediction_single(stopID string, direction int, parentStationID string) (PredictionData, string, error) {
	// constructing the request.
	// stopID is the individual platform ID (e.g., "70106"), parentStationID is for storage (e.g., "place-lake")
	// Fetch predictions for Green-B only to avoid mixing with C/D/E trains on shared downtown platforms.
	// Keep the same page limit for now to minimize API load while improving relevance.
	url := fmt.Sprintf("https://api-v3.mbta.com/predictions?filter[stop]=%s&filter[direction_id]=%d&filter[route]=Green-B&sort=arrival_time&page[limit]=2", stopID, direction)
	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return PredictionData{}, "nil", err
	}
	req.Header.Set("x-api-key", key)
	client := &http.Client{Timeout: 10 * time.Second}

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

	// Check if we have any predictions
	if len(predictions) == 0 {
		// No predictions available - this is normal during off-peak hours
		return PredictionData{}, "nil", fmt.Errorf("no predictions available")
	}

	// Store ALL predictions under the parent station ID for matching with arrivals
	// Use parentStationID if provided, otherwise fall back to stopID
	storageKey := parentStationID
	if storageKey == "" {
		storageKey = stopID
	}
	storageKey = normalizeStationKey(storageKey)

	predictionMutex.Lock()
	if predictionDataMap[storageKey] == nil {
		predictionDataMap[storageKey] = make(map[int]map[string][]PredictionData)
	}
	if predictionDataMap[storageKey][direction] == nil {
		predictionDataMap[storageKey][direction] = make(map[string][]PredictionData)
	}

	// Store each prediction by its vehicle ID
	storedCount := 0
	for _, pred := range predictions {
		if pred.VehicleID != "" && pred.ArrivalTime != nil {
			predictionDataMap[storageKey][direction][pred.VehicleID] = append(
				predictionDataMap[storageKey][direction][pred.VehicleID],
				pred,
			)
			debugf("Stored prediction: Platform %s Station %s, Dir %d, Vehicle %s (Arrival: %v)\n",
				stopID, storageKey, direction, pred.VehicleID, pred.ArrivalTime)
			storedCount++
		}
	}
	predictionMutex.Unlock()

	if storedCount == 0 {
		return PredictionData{}, "nil", fmt.Errorf("no valid predictions with arrival times")
	}

	// Return the second prediction if available for backward compatibility
	var returnPred PredictionData
	var returnVehicle string
	if len(predictions) >= 2 {
		returnPred = predictions[1]
		returnVehicle = predictions[1].VehicleID
	} else {
		returnPred = predictions[0]
		returnVehicle = predictions[0].VehicleID
	}

	return returnPred, returnVehicle, nil
}

// cleanupOldPredictions removes predictions older than 2 hours to prevent memory leak
func cleanupOldPredictions() {
	predictionMutex.Lock()
	defer predictionMutex.Unlock()

	cutoffTime := time.Now().Add(-120 * time.Minute)
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
		fmt.Printf("Cleaned up %d old predictions (older than 2 hours)\n", cleanedCount)
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

// Initialize all Green-B stations with zero values in the accuracy maps
func initializeStationMaps() {
	statsMutex.Lock()
	defer statsMutex.Unlock()

	for stationID, stationName := range greenBStationNames {
		// Initialize stationAccuracyMap if needed
		if stationAccuracyMap[stationID] == nil {
			stationAccuracyMap[stationID] = make(map[int]*StationStats)
		}
		// Initialize both directions with zero values
		if stationAccuracyMap[stationID][0] == nil {
			stationAccuracyMap[stationID][0] = &StationStats{} // Outbound
		}
		if stationAccuracyMap[stationID][1] == nil {
			stationAccuracyMap[stationID][1] = &StationStats{} // Inbound
		}

		// Initialize cached accuracy response for this station
		cachedAccuracyMap[stationID] = &StationAccuracyResponse{
			StationID:        stationID,
			StationName:      stationName,
			InboundAccuracy:  0.0,
			InboundTotal:     0,
			OutboundAccuracy: 0.0,
			OutboundTotal:    0,
		}
	}

	fmt.Printf("??Initialized accuracy maps for %d Green-B stations\n", len(greenBStationNames))
}

// Update cached accuracy for a specific station (call this after updating stats)
func updateCachedAccuracy(stationID string) {
	// Must be called with statsMutex already locked
	if cachedAccuracyMap[stationID] == nil {
		cachedAccuracyMap[stationID] = &StationAccuracyResponse{
			StationID:   stationID,
			StationName: greenBStationNames[stationID],
		}
		if cachedAccuracyMap[stationID].StationName == "" {
			cachedAccuracyMap[stationID].StationName = stationID
		}
	}

	cached := cachedAccuracyMap[stationID]

	// Update inbound stats (direction 1)
	if stats, exists := stationAccuracyMap[stationID][1]; exists {
		cached.InboundTotal = stats.Total
		if stats.Total > 0 {
			cached.InboundAccuracy = float64(stats.Correct) / float64(stats.Total) * 100
		} else {
			cached.InboundAccuracy = 0.0
		}
	}

	// Update outbound stats (direction 0)
	if stats, exists := stationAccuracyMap[stationID][0]; exists {
		cached.OutboundTotal = stats.Total
		if stats.Total > 0 {
			cached.OutboundAccuracy = float64(stats.Correct) / float64(stats.Total) * 100
		} else {
			cached.OutboundAccuracy = 0.0
		}
	}

	// Debug log to verify cache update
	debugf("Updated cache for %s: Inbound=%.2f%% (%d total), Outbound=%.2f%% (%d total)\n",
		stationID, cached.InboundAccuracy, cached.InboundTotal, cached.OutboundAccuracy, cached.OutboundTotal)
}

// HTTP handler for debug/health check
func handleDebug(w http.ResponseWriter, r *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	predictionMutex.RLock()
	statsMutex.Lock()

	// Count predictions
	predictionCount := 0
	for _, directions := range predictionDataMap {
		for _, vehicles := range directions {
			for _, predictions := range vehicles {
				predictionCount += len(predictions)
			}
		}
	}

	// Count stations with stats
	stationCount := len(stationAccuracyMap)
	totalArrivals := 0
	for _, directions := range stationAccuracyMap {
		for _, stats := range directions {
			totalArrivals += stats.Total
		}
	}

	statsMutex.Unlock()
	predictionMutex.RUnlock()

	response := map[string]interface{}{
		"status":            "running",
		"api_key_set":       key != "",
		"predictions_count": predictionCount,
		"stations_tracked":  stationCount,
		"total_arrivals":    totalArrivals,
		"timestamp":         time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
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

	// Return all stations from the cached map (includes stations with 0 arrivals)
	response := make([]StationAccuracyResponse, 0, len(cachedAccuracyMap))
	nonZeroCount := 0
	for _, stationData := range cachedAccuracyMap {
		response = append(response, *stationData)
		if stationData.InboundTotal > 0 || stationData.OutboundTotal > 0 {
			nonZeroCount++
		}
	}

	// Debug log to show what we're returning
	debugf("/api/statistics called: Returning %d stations (%d with data)\n", len(response), nonZeroCount)
	if nonZeroCount > 0 {
		// Show first station with non-zero data for debugging
		for _, station := range response {
			if station.InboundTotal > 0 || station.OutboundTotal > 0 {
				debugf("Sample: %s (%s) - In: %.2f%% (%d), Out: %.2f%% (%d)\n",
					station.StationID, station.StationName,
					station.InboundAccuracy, station.InboundTotal,
					station.OutboundAccuracy, station.OutboundTotal)
				break
			}
		}
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
		pred, _, err := fetchPrediction_single("70135", 0, "place-brico") // example stop ID (Packards Corner)
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
	// Initialize all Green-B stations with zero values
	initializeStationMaps()

	// Get port from environment variable or use 8080 as default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start HTTP API server for frontend
	go func() {
		http.HandleFunc("/api/statistics", handleGetStatistics)
		http.HandleFunc("/api/debug", handleDebug)
		addr := "0.0.0.0:" + port
		fmt.Printf("HTTP API server starting on port %s (accessible at http://0.0.0.0:%s)\n", port, port)
		if err := http.ListenAndServe(addr, nil); err != nil {
			fmt.Printf("HTTP server error: %v\n", err)
			panic(err) // Crash if server fails to start
		}
	}()

	// Start periodic cleanup of old predictions (every 180 minutes)
	go func() {
		cleanupTicker := time.NewTicker(180 * time.Minute)
		defer cleanupTicker.Stop()
		fmt.Println("Started periodic prediction cleanup (every 180 minutes)...")
		for {
			<-cleanupTicker.C
			cleanupOldPredictions()
		}
	}()

	// Keep-alive: Prevent Render.com from spinning down the server
	// Ping ourselves every 14 minutes to maintain activity
	go func() {
		keepAliveTicker := time.NewTicker(10 * time.Minute)
		defer keepAliveTicker.Stop()

		// Get backend URL from environment or construct from PORT
		backendURL := os.Getenv("RENDER_EXTERNAL_URL")
		if backendURL == "" {
			// Fallback: try to construct URL (works if RENDER env vars are available)
			backendURL = "http://localhost:" + port
		}

		fmt.Printf("Started keep-alive service (pinging %s every 14 minutes)...\n", backendURL)

		for {
			<-keepAliveTicker.C
			// Make a lightweight GET request to keep the server active
			resp, err := http.Get(backendURL + "/api/statistics")
			if err != nil {
				fmt.Printf("Keep-alive ping failed: %v\n", err)
			} else {
				resp.Body.Close()
				fmt.Printf("Keep-alive ping successful at %s\n", time.Now().Format(time.RFC3339))
			}
		}
	}()

	// Reset statistics daily at 3:30 AM America/New_York.
	go func() {
		location, err := time.LoadLocation("America/New_York")
		if err != nil {
			fmt.Printf("Warning: Could not load America/New_York timezone, falling back to local time: %v\n", err)
			location = time.Local
		}

		resetStats := func() {
			// Flush prediction cache.
			predictionMutex.Lock()
			predictionStations := len(predictionDataMap)
			predictionDataMap = make(map[string]map[int]map[string][]PredictionData)
			predictionMutex.Unlock()

			// Flush runtime vehicle/stop caches.
			mapMutex.Lock()
			vehicleCount := len(actualTrainInfo)
			nextStopCount := len(vehicleNextStop)
			dynamicStopCount := len(dynamicStopGeoLocation)
			stopParentCount := len(stopToParentStation)
			arrivalSentCount := len(vehicleStopArrivalSent)

			actualTrainInfo = make(map[string]ActualData)
			vehicleNextStop = make(map[string]string)
			dynamicStopGeoLocation = make(map[string][2]float64)
			stopToParentStation = make(map[string]string)
			vehicleStopArrivalSent = make(map[string]bool)
			mapMutex.Unlock()

			// Drain queued arrivals so old events are not graded after reset.
			drainedArrivals := 0
			for {
				select {
				case <-ArrivalChannel:
					drainedArrivals++
				default:
					goto drained
				}
			}
		drained:

			// Flush accuracy stats/cache.
			statsMutex.Lock()
			totalPredictions := 0
			for _, directions := range stationAccuracyMap {
				for _, stats := range directions {
					totalPredictions += stats.Total
				}
			}
			stationAccuracyMap = make(map[string]map[int]*StationStats)
			cachedAccuracyMap = make(map[string]*StationAccuracyResponse)
			statsMutex.Unlock()

			fmt.Printf("Full in-memory reset completed at %s | predictionStations=%d totalPredictions=%d vehicles=%d nextStops=%d dynamicStops=%d stopParents=%d arrivalSent=%d drainedArrivals=%d\n",
				time.Now().In(location).Format(time.RFC3339),
				predictionStations,
				totalPredictions,
				vehicleCount,
				nextStopCount,
				dynamicStopCount,
				stopParentCount,
				arrivalSentCount,
				drainedArrivals,
			)
		}

		now := time.Now().In(location)
		nextReset := time.Date(now.Year(), now.Month(), now.Day(), 3, 30, 0, 0, location)
		if !now.Before(nextReset) {
			nextReset = nextReset.Add(24 * time.Hour)
		}
		initialWait := nextReset.Sub(now)

		fmt.Printf("Started daily statistics reset at 03:30 (%s). First reset in %s at %s\n",
			location.String(), initialWait.Round(time.Second), nextReset.Format(time.RFC3339))

		timer := time.NewTimer(initialWait)
		defer timer.Stop()

		<-timer.C
		resetStats()

		resetTicker := time.NewTicker(24 * time.Hour)
		defer resetTicker.Stop()
		for {
			<-resetTicker.C
			resetStats()
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
			debugln("\n=== TRAIN ARRIVAL DETECTED ===")
			debugf("Station (Place ID): %s\n", arrival.StationPlaceID)
			debugf("Station (Stop ID): %s\n", arrival.StationStopID)
			debugf("Train ID: %s\n", arrival.TrainID)
			debugf("Direction: %d\n", arrival.Direction)
			debugf("Actual Arrival Time: %s\n", arrival.ArrivalTime.Format(time.RFC3339))
			debugln("==============================")

			// Compare with prediction data and score accuracy
			// Use StationPlaceID (parent station) to match how predictions are stored
			predictionMutex.RLock()
			debugf("Looking up predictions for Place ID: %s, Direction: %d, Train: %s\n", arrival.StationPlaceID, arrival.Direction, arrival.TrainID)

			// Debug: Show what's in predictionDataMap for this station
			if stationPreds, hasStation := predictionDataMap[arrival.StationPlaceID]; hasStation {
				if dirPreds, hasDir := stationPreds[arrival.Direction]; hasDir {
					debugf("  Found direction %d data with %d vehicles: %v\n", arrival.Direction, len(dirPreds), func() []string {
						keys := make([]string, 0, len(dirPreds))
						for k := range dirPreds {
							keys = append(keys, k)
						}
						return keys
					}())
				} else {
					debugf("  No predictions for direction %d\n", arrival.Direction)
				}
			} else {
				debugf("  No predictions stored for station %s\n", arrival.StationPlaceID)
			}

			if predictions, exists := predictionDataMap[arrival.StationPlaceID][arrival.Direction][arrival.TrainID]; exists {
				debugf("Found %d prediction(s) for this train:\n", len(predictions))
				targetObservationTime := arrival.ArrivalTime.Add(-5 * time.Minute)
				gradedCount := 0
				correctCount := 0
				incorrectCount := 0

				// Grade every prediction observed at least 5 minutes before actual arrival.
				for i := range predictions {
					pred := &predictions[i]
					if pred.ArrivalTime == nil {
						continue
					}
					if pred.ObservationTime.After(targetObservationTime) {
						continue
					}

					difference := arrival.ArrivalTime.Sub(*pred.ArrivalTime)
					diffSeconds := difference.Seconds()
					diffMinutes := difference.Minutes()

					debugf("\nGraded prediction #%d (observed <= arrival-5m):\n", i+1)
					debugf("  Snapshot Time:     %s\n", pred.ObservationTime.Format(time.RFC3339))
					debugf("  Target Time:       %s\n", targetObservationTime.Format(time.RFC3339))
					debugf("  Predicted Arrival: %s\n", pred.ArrivalTime.Format(time.RFC3339))
					debugf("  Actual Arrival:    %s\n", arrival.ArrivalTime.Format(time.RFC3339))
					debugf("  Difference:        %.0f seconds (%.2f minutes)\n", diffSeconds, diffMinutes)

					isCorrect := math.Abs(diffMinutes) <= 3
					if isCorrect {
						correctCount++
					} else {
						incorrectCount++
					}
					gradedCount++

					if diffSeconds > 0 {
						debugf("  Status:            Train arrived %.2f minutes LATE\n", diffMinutes)
					} else if diffSeconds < 0 {
						debugf("  Status:            Train arrived %.2f minutes EARLY\n", math.Abs(diffMinutes))
					} else {
						debugln("  Status:            Train arrived EXACTLY on time!")
					}
				}

				if gradedCount > 0 {
					// Update statistics for this arrival with all eligible predictions.
					statsMutex.Lock()
					if stationAccuracyMap[arrival.StationPlaceID] == nil {
						stationAccuracyMap[arrival.StationPlaceID] = make(map[int]*StationStats)
					}
					if stationAccuracyMap[arrival.StationPlaceID][arrival.Direction] == nil {
						stationAccuracyMap[arrival.StationPlaceID][arrival.Direction] = &StationStats{}
					}
					stats := stationAccuracyMap[arrival.StationPlaceID][arrival.Direction]
					stats.Total += gradedCount
					stats.Correct += correctCount
					stats.Incorrect += incorrectCount

					updateCachedAccuracy(arrival.StationPlaceID)
					accuracy := float64(stats.Correct) / float64(stats.Total) * 100
					directionName := "Outbound"
					if arrival.Direction == 1 {
						directionName = "Inbound"
					}
					debugf("\nSTATION STATISTICS [%s - %s]:\n", arrival.StationPlaceID, directionName)
					debugf("   Total Predictions: %d\n", stats.Total)
					debugf("   Correct (<=3 min): %d\n", stats.Correct)
					debugf("   Wrong (>3 min):   %d\n", stats.Incorrect)
					debugf("   Accuracy Rate:    %.2f%%\n", accuracy)
					statsMutex.Unlock()
				} else {
					debugln("No predictions observed at least 5 minutes before arrival")
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
				debugf("Cleaned up predictions for vehicle %s at station %s\n", arrival.TrainID, arrival.StationPlaceID)
			} else {
				debugln("No predictions found for this train arrival")
				predictionMutex.RUnlock()
			}

			debugln("\n==============================")
		}
	}()

	// Start goroutine to fetch predictions every 1 minutes
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		debugln("Started periodic prediction fetching (every 1 minutes)...")

		// Map of platform stops to their parent stations
		// Format: platformID -> parentStationID
		platformToParent := map[string]string{
			"70106": "place-lake", "70107": "place-lake", // Boston College
			"70110": "place-sougr", "70111": "place-sougr", // South Street
			"70112": "place-chill", "70113": "place-chill", // Chestnut Hill Avenue
			"70114": "place-chswk", "70115": "place-chswk", // Chiswick Road
			"70116": "place-sthld", "70117": "place-sthld", // Sutherland Road
			"70120": "place-wascm", "70121": "place-wascm", // Washington Street
			"70124": "place-wrnst", "70125": "place-wrnst", // Warren Street
			"70126": "place-alsgr", "70127": "place-alsgr", // Allston Street
			"70128": "place-grigg", "70129": "place-grigg", // Griggs Street
			"70130": "place-harvd", "70131": "place-harvd", // Harvard Avenue
			"70134": "place-brico", "70135": "place-brico", // Packards Corner
			// Canonicalize orphan-platform station keys so both directions aggregate to one station ID.
			"70136": "70136", "70137": "70136", // Babcock Street (legacy/orphan IDs)
			"170136": "70136", "170137": "70136", // Babcock Street current IDs
			"70140": "70140", "70141": "70140", // Amory Street (legacy/orphan IDs)
			"170140": "70140", "170141": "70140", // Amory Street current IDs (Saint Paul Street B)
			"70144": "place-bucen", "70145": "place-bucen", // Boston University Central
			"70146": "place-buest", "70147": "place-buest", // Boston University East
			"70148": "place-bland", "70149": "place-bland", // Blandford Street
			"70150": "place-kencl", "70151": "place-kencl", "71150": "place-kencl", "71151": "place-kencl", // Kenmore
			"70152": "place-hymnl", "70153": "place-hymnl", // Hynes Convention Center
			"70154": "place-coecl", "70155": "place-coecl", // Copley
			"70156": "place-armnl", "70157": "place-armnl", // Arlington
			"70158": "place-boyls", "70159": "place-boyls", // Boylston
			"70196": "place-pktrm", "70197": "place-pktrm", "70198": "place-pktrm", "70199": "place-pktrm", "70200": "place-pktrm", "71199": "place-pktrm", // Park Street
			"70201": "place-gover", "70202": "place-gover", // Government Center
		}

		// Helper function to fetch predictions for a platform
		fetchForPlatform := func(platformID, parentID string) {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Recovered from panic in fetchPrediction_single for platform %s: %v\n", platformID, r)
				}
			}()

			// Fetch both directions
			if _, _, err := fetchPrediction_single(platformID, 0, parentID); err != nil {
				if err.Error() != "no predictions available" {
					fmt.Printf("Error fetching prediction for platform %s direction 0: %v\n", platformID, err)
				}
			}
			if _, _, err := fetchPrediction_single(platformID, 1, parentID); err != nil {
				if err.Error() != "no predictions available" {
					fmt.Printf("Error fetching prediction for platform %s direction 1: %v\n", platformID, err)
				}
			}
		}

		// Fetch predictions immediately on startup
		debugln("Fetching initial predictions for all platforms...")
		for platformID, parentID := range platformToParent {
			go fetchForPlatform(platformID, parentID)
		}

		// Then continue with periodic updates
		for {
			<-ticker.C
			debugln("Fetching updated predictions...")
			for platformID, parentID := range platformToParent {
				go fetchForPlatform(platformID, parentID)
			}
		}
	}()

	// Block forever to keep the program running while goroutines work
	select {}
}

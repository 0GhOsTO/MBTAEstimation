// Requirement before running: create go module
// cd f:\CS\ProjectGithub\MBTAEstimation\backend; go mod init mbta-backend
// Create .env file in backend folder with content:
// MBTA_API_KEY=your_actual_api_key_here
// MBTA_API_KEY_C=your_green_c_api_key_here
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

const (
	routeGreenB = "Green-B"
	routeGreenC = "Green-C"
	routeGreenD = "Green-D"
	routeGreenE = "Green-E"
)

var requiredEnvByRoute = map[string][]string{
	routeGreenB: []string{"MBTA_API_KEY"},
	routeGreenC: []string{"MBTA_API_KEY_C", "MBTA_API_KEY_c"},
	routeGreenD: []string{"MBTA_API_KEY_D"},
	routeGreenE: []string{"MBTA_API_KEY_E"},
}

var routeAPIKeys = make(map[string]string)

// stop id: {in/outbound : {vehicle id: prediction data}}}
// Map structure: route -> stationID -> direction -> vehicleID -> []PredictionData
var predictionDataMap = make(map[string]map[string]map[int]map[string][]PredictionData)

// mutex for protecting prediction map from concurrent access
var predictionMutex sync.RWMutex

// StationStats holds accuracy statistics for a station and direction
type StationStats struct {
	Total     int
	Correct   int
	Incorrect int
}

// Prediction accuracy statistics by station and direction
// Map structure: route -> stationID -> direction (0=outbound, 1=inbound) -> stats
var stationAccuracyMap = make(map[string]map[string]map[int]*StationStats)
var statsMutex sync.Mutex

// Cached accuracy rates for quick API responses
// Map structure: route -> stationID -> StationAccuracyResponse (with calculated accuracy)
var cachedAccuracyMap = make(map[string]map[string]*StationAccuracyResponse)

// Most recent arrival prediction difference by station and direction.
// Map structure: route -> stationID -> direction (0=outbound, 1=inbound) -> diff metadata
var stationRecentDiffMap = make(map[string]map[string]map[int]*RecentPredictionDiff)

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
	"place-brico": "Packard's Corner",
	"place-babck": "Babcock Street",
	"place-amory": "Amory Street",
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

var greenCStationNames = map[string]string{
	"place-clmnl": "Cleveland Circle",
	"place-engav": "Englewood Avenue",
	"place-denrd": "Dean Road",
	"place-tapst": "Tappan Street",
	"place-bcnwa": "Washington Square",
	"place-fbkst": "Fairbanks Street",
	"place-bndhl": "Brandon Hall",
	"place-sumav": "Summit Avenue",
	"place-cool":  "Coolidge Corner",
	"place-stpul": "Saint Paul Street",
	"place-kntst": "Kent Street",
	"place-hwsst": "Hawes Street",
	"place-smary": "Saint Mary's Street",
	"place-kencl": "Kenmore",
	"place-hymnl": "Hynes Convention Center",
	"place-coecl": "Copley",
	"place-armnl": "Arlington",
	"place-boyls": "Boylston",
	"place-pktrm": "Park Street",
	"place-gover": "Government Center",
}

var greenDStationNames = map[string]string{
	"place-river": "Riverside",
	"place-woodl": "Woodland",
	"place-waban": "Waban",
	"place-eliot": "Eliot",
	"place-newtn": "Newton Highlands",
	"place-newto": "Newton Centre",
	"place-chhil": "Chestnut Hill",
	"place-rsmnl": "Reservoir",
	"place-bcnfd": "Beaconsfield",
	"place-brkhl": "Brookline Hills",
	"place-bvmnl": "Brookline Village",
	"place-longw": "Longwood",
	"place-fenwy": "Fenway",
	"place-kencl": "Kenmore",
	"place-hymnl": "Hynes Convention Center",
	"place-coecl": "Copley",
	"place-armnl": "Arlington",
	"place-boyls": "Boylston",
	"place-pktrm": "Park Street",
	"place-gover": "Government Center",
	"place-haecl": "Haymarket",
	"place-north": "North Station",
	"place-spmnl": "Science Park/West End",
	"place-lech":  "Lechmere",
	"place-unsqu": "Union Square",
}

var greenEStationNames = map[string]string{
	"place-hsmnl": "Heath Street",
	"place-bckhl": "Back of the Hill",
	"place-rvrwy": "Riverway",
	"place-mispk": "Mission Park",
	"place-fenwd": "Fenwood Road",
	"place-brmnl": "Brigham Circle",
	"place-lngmd": "Longwood Medical Area",
	"place-mfa":   "Museum of Fine Arts",
	"place-nuniv": "Northeastern University",
	"place-symcl": "Symphony",
	"place-prmnl": "Prudential",
	"place-coecl": "Copley",
	"place-armnl": "Arlington",
	"place-boyls": "Boylston",
	"place-pktrm": "Park Street",
	"place-gover": "Government Center",
	"place-haecl": "Haymarket",
	"place-north": "North Station",
	"place-spmnl": "Science Park/West End",
	"place-lech":  "Lechmere",
	"place-esomr": "East Somerville",
	"place-gilmn": "Gilman Square",
	"place-mgngl": "Magoun Square",
	"place-balsq": "Ball Square",
	"place-mdftf": "Medford/Tufts",
}

var stationNamesByRoute = map[string]map[string]string{
	routeGreenB: greenBStationNames,
	routeGreenC: greenCStationNames,
	routeGreenD: greenDStationNames,
	routeGreenE: greenEStationNames,
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

func loadRouteAPIKeysFromEnv(lookup func(string) string) (map[string]string, error) {
	keys := make(map[string]string, len(requiredEnvByRoute))
	for route, envVars := range requiredEnvByRoute {
		key := ""
		for _, envVar := range envVars {
			key = strings.TrimSpace(lookup(envVar))
			if key != "" {
				break
			}
		}
		if key == "" {
			return nil, fmt.Errorf("%s environment variable not set", strings.Join(envVars, " or "))
		}
		keys[route] = key
	}
	return keys, nil
}

func normalizeRoute(route string) (string, error) {
	trimmed := strings.TrimSpace(route)
	if trimmed == "" {
		return routeGreenB, nil
	}
	switch trimmed {
	case routeGreenB, routeGreenC, routeGreenD, routeGreenE:
		return trimmed, nil
	default:
		return "", fmt.Errorf("unsupported route: %s", route)
	}
}

func apiKeyForRoute(route string) (string, error) {
	normalizedRoute, err := normalizeRoute(route)
	if err != nil {
		return "", err
	}
	key, ok := routeAPIKeys[normalizedRoute]
	if !ok || strings.TrimSpace(key) == "" {
		return "", fmt.Errorf("no API key configured for route %s", normalizedRoute)
	}
	return key, nil
}

func routeNamesForStats(route string) map[string]string {
	if names, ok := stationNamesByRoute[route]; ok {
		return names
	}
	return map[string]string{}
}

func init() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("No .env file found (this is normal in production)")
	}
	keys, err := loadRouteAPIKeysFromEnv(os.Getenv)
	if err != nil {
		panic(err)
	}
	routeAPIKeys = keys
	fmt.Printf("MBTA API keys loaded successfully for routes: %s, %s, %s, %s\n", routeGreenB, routeGreenC, routeGreenD, routeGreenE)
}

func fetchPredictionSingle(route, stopID string, direction int, parentStationID string) (PredictionData, string, error) {
	// constructing the request.
	// stopID is the individual platform ID (e.g., "70106"), parentStationID is for storage (e.g., "place-lake")
	normalizedRoute, err := normalizeRoute(route)
	if err != nil {
		return PredictionData{}, "nil", err
	}
	apiKey, err := apiKeyForRoute(normalizedRoute)
	if err != nil {
		return PredictionData{}, "nil", err
	}
	// Keep the same page limit for now to minimize API load while improving relevance.
	url := fmt.Sprintf("https://api-v3.mbta.com/predictions?filter[stop]=%s&filter[direction_id]=%d&filter[route]=%s&sort=arrival_time&page[limit]=2", stopID, direction, normalizedRoute)
	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return PredictionData{}, "nil", err
	}
	req.Header.Set("x-api-key", apiKey)
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
	if predictionDataMap[normalizedRoute] == nil {
		predictionDataMap[normalizedRoute] = make(map[string]map[int]map[string][]PredictionData)
	}
	if predictionDataMap[normalizedRoute][storageKey] == nil {
		predictionDataMap[normalizedRoute][storageKey] = make(map[int]map[string][]PredictionData)
	}
	if predictionDataMap[normalizedRoute][storageKey][direction] == nil {
		predictionDataMap[normalizedRoute][storageKey][direction] = make(map[string][]PredictionData)
	}

	// Store each prediction by its vehicle ID
	storedCount := 0
	for _, pred := range predictions {
		if pred.VehicleID != "" && pred.ArrivalTime != nil {
			predictionDataMap[normalizedRoute][storageKey][direction][pred.VehicleID] = append(
				predictionDataMap[normalizedRoute][storageKey][direction][pred.VehicleID],
				pred,
			)
			debugf("Stored prediction: Route %s Platform %s Station %s, Dir %d, Vehicle %s (Arrival: %v)\n",
				normalizedRoute, stopID, storageKey, direction, pred.VehicleID, pred.ArrivalTime)
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

	for route, stations := range predictionDataMap {
		for stopID, directions := range stations {
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
						predictionDataMap[route][stopID][direction][vehicleID] = validPredictions
					} else {
						delete(predictionDataMap[route][stopID][direction], vehicleID)
					}
				}
				// Clean up empty direction maps
				if len(predictionDataMap[route][stopID][direction]) == 0 {
					delete(predictionDataMap[route][stopID], direction)
				}
			}
			// Clean up empty stop maps
			if len(predictionDataMap[route][stopID]) == 0 {
				delete(predictionDataMap[route], stopID)
			}
		}
		// Clean up empty route maps
		if len(predictionDataMap[route]) == 0 {
			delete(predictionDataMap, route)
		}
	}

	if cleanedCount > 0 {
		fmt.Printf("Cleaned up %d old predictions (older than 2 hours)\n", cleanedCount)
	}
}

// StationAccuracyResponse represents the JSON response for station accuracy data
type StationAccuracyResponse struct {
	StationID                 string   `json:"station_id"`
	StationName               string   `json:"station_name"`
	InboundAccuracy           float64  `json:"inbound_accuracy"`
	InboundTotal              int      `json:"inbound_total"`
	OutboundAccuracy          float64  `json:"outbound_accuracy"`
	OutboundTotal             int      `json:"outbound_total"`
	InboundRecentDiffMinutes  *float64 `json:"inbound_recent_diff_minutes"`
	OutboundRecentDiffMinutes *float64 `json:"outbound_recent_diff_minutes"`
}

// RecentPredictionDiff stores the latest arrival-vs-prediction delta for a station/direction.
type RecentPredictionDiff struct {
	DifferenceMinutes         float64
	ArrivalTime               time.Time
	PredictionObservationTime time.Time
	PredictionArrivalTime     time.Time
	TrainID                   string
}

// selectMiddlePrediction chooses the middle prediction by observation time.
// For even counts, it selects the older of the two middle entries.
func selectMiddlePrediction(predictions []PredictionData, arrivalTime time.Time) (PredictionData, bool) {
	valid := make([]PredictionData, 0, len(predictions))
	for _, pred := range predictions {
		if pred.ArrivalTime == nil {
			continue
		}
		// Ignore snapshots observed after actual arrival when possible.
		if pred.ObservationTime.After(arrivalTime) {
			continue
		}
		valid = append(valid, pred)
	}

	// Fallback: if all snapshots are after actual arrival, still use available predictions.
	if len(valid) == 0 {
		for _, pred := range predictions {
			if pred.ArrivalTime != nil {
				valid = append(valid, pred)
			}
		}
	}

	if len(valid) == 0 {
		return PredictionData{}, false
	}

	sort.Slice(valid, func(i, j int) bool {
		if valid[i].ObservationTime.Equal(valid[j].ObservationTime) {
			return valid[i].ArrivalTime.Before(*valid[j].ArrivalTime)
		}
		return valid[i].ObservationTime.Before(valid[j].ObservationTime)
	})

	middleIndex := (len(valid) - 1) / 2
	return valid[middleIndex], true
}

// Initialize configured stations with zero values in the accuracy maps.
func initializeStationMaps() {
	statsMutex.Lock()
	defer statsMutex.Unlock()

	for route, stationNames := range stationNamesByRoute {
		if stationAccuracyMap[route] == nil {
			stationAccuracyMap[route] = make(map[string]map[int]*StationStats)
		}
		if cachedAccuracyMap[route] == nil {
			cachedAccuracyMap[route] = make(map[string]*StationAccuracyResponse)
		}
		if stationRecentDiffMap[route] == nil {
			stationRecentDiffMap[route] = make(map[string]map[int]*RecentPredictionDiff)
		}
		for stationID, stationName := range stationNames {
			if stationAccuracyMap[route][stationID] == nil {
				stationAccuracyMap[route][stationID] = make(map[int]*StationStats)
			}
			if stationAccuracyMap[route][stationID][0] == nil {
				stationAccuracyMap[route][stationID][0] = &StationStats{} // Outbound
			}
			if stationAccuracyMap[route][stationID][1] == nil {
				stationAccuracyMap[route][stationID][1] = &StationStats{} // Inbound
			}

			cachedAccuracyMap[route][stationID] = &StationAccuracyResponse{
				StationID:                 stationID,
				StationName:               stationName,
				InboundAccuracy:           0.0,
				InboundTotal:              0,
				OutboundAccuracy:          0.0,
				OutboundTotal:             0,
				InboundRecentDiffMinutes:  nil,
				OutboundRecentDiffMinutes: nil,
			}
		}
		fmt.Printf("Initialized accuracy maps for %d stations on %s\n", len(stationNames), route)
	}
}

// Update cached accuracy for a specific route/station (call this after updating stats).
func updateCachedAccuracy(route, stationID string) {
	// Must be called with statsMutex already locked
	if cachedAccuracyMap[route] == nil {
		cachedAccuracyMap[route] = make(map[string]*StationAccuracyResponse)
	}
	if stationAccuracyMap[route] == nil {
		stationAccuracyMap[route] = make(map[string]map[int]*StationStats)
	}
	if stationRecentDiffMap[route] == nil {
		stationRecentDiffMap[route] = make(map[string]map[int]*RecentPredictionDiff)
	}
	if cachedAccuracyMap[route][stationID] == nil {
		cachedAccuracyMap[route][stationID] = &StationAccuracyResponse{
			StationID:   stationID,
			StationName: routeNamesForStats(route)[stationID],
		}
		if cachedAccuracyMap[route][stationID].StationName == "" {
			cachedAccuracyMap[route][stationID].StationName = stationID
		}
	}

	cached := cachedAccuracyMap[route][stationID]

	// Update inbound stats (direction 1)
	if stats, exists := stationAccuracyMap[route][stationID][1]; exists {
		cached.InboundTotal = stats.Total
		if stats.Total > 0 {
			cached.InboundAccuracy = float64(stats.Correct) / float64(stats.Total) * 100
		} else {
			cached.InboundAccuracy = 0.0
		}
	}

	// Update outbound stats (direction 0)
	if stats, exists := stationAccuracyMap[route][stationID][0]; exists {
		cached.OutboundTotal = stats.Total
		if stats.Total > 0 {
			cached.OutboundAccuracy = float64(stats.Correct) / float64(stats.Total) * 100
		} else {
			cached.OutboundAccuracy = 0.0
		}
	}

	// Update most recent arrival prediction difference (direction 1 = inbound)
	if dirData, exists := stationRecentDiffMap[route][stationID]; exists {
		if recent, hasInbound := dirData[1]; hasInbound && recent != nil {
			diff := recent.DifferenceMinutes
			cached.InboundRecentDiffMinutes = &diff
		} else {
			cached.InboundRecentDiffMinutes = nil
		}
		if recent, hasOutbound := dirData[0]; hasOutbound && recent != nil {
			diff := recent.DifferenceMinutes
			cached.OutboundRecentDiffMinutes = &diff
		} else {
			cached.OutboundRecentDiffMinutes = nil
		}
	} else {
		cached.InboundRecentDiffMinutes = nil
		cached.OutboundRecentDiffMinutes = nil
	}

	// Debug log to verify cache update
	debugf("Updated cache for %s/%s: Inbound=%.2f%% (%d total), Outbound=%.2f%% (%d total)\n",
		route, stationID, cached.InboundAccuracy, cached.InboundTotal, cached.OutboundAccuracy, cached.OutboundTotal)
}

func statisticsRouteFromRequest(r *http.Request) (string, error) {
	rawRoute := strings.TrimSpace(r.URL.Query().Get("route"))
	if rawRoute == "" {
		return routeGreenB, nil // Backward compatibility default.
	}
	return normalizeRoute(rawRoute)
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

	predictionsByRoute := map[string]int{
		routeGreenB: 0,
		routeGreenC: 0,
		routeGreenD: 0,
		routeGreenE: 0,
	}
	stationsByRoute := map[string]int{
		routeGreenB: 0,
		routeGreenC: 0,
		routeGreenD: 0,
		routeGreenE: 0,
	}
	arrivalsByRoute := map[string]int{
		routeGreenB: 0,
		routeGreenC: 0,
		routeGreenD: 0,
		routeGreenE: 0,
	}
	totalPredictionCount := 0
	totalStationsTracked := 0
	totalArrivals := 0

	for route, stations := range predictionDataMap {
		for _, directions := range stations {
			for _, vehicles := range directions {
				for _, predictions := range vehicles {
					predictionsByRoute[route] += len(predictions)
					totalPredictionCount += len(predictions)
				}
			}
		}
	}
	for route, stations := range stationAccuracyMap {
		stationsByRoute[route] = len(stations)
		totalStationsTracked += len(stations)
		for _, directions := range stations {
			for _, stats := range directions {
				arrivalsByRoute[route] += stats.Total
				totalArrivals += stats.Total
			}
		}
	}

	statsMutex.Unlock()
	predictionMutex.RUnlock()

	keysLoaded := map[string]bool{
		routeGreenB: strings.TrimSpace(routeAPIKeys[routeGreenB]) != "",
		routeGreenC: strings.TrimSpace(routeAPIKeys[routeGreenC]) != "",
		routeGreenD: strings.TrimSpace(routeAPIKeys[routeGreenD]) != "",
		routeGreenE: strings.TrimSpace(routeAPIKeys[routeGreenE]) != "",
	}

	response := map[string]interface{}{
		"status":                   "running",
		"routes_active":            []string{routeGreenB, routeGreenC, routeGreenD, routeGreenE},
		"api_keys_loaded_by_route": keysLoaded,
		"predictions_count":        totalPredictionCount,
		"predictions_by_route":     predictionsByRoute,
		"stations_tracked":         totalStationsTracked,
		"stations_by_route":        stationsByRoute,
		"total_arrivals":           totalArrivals,
		"arrivals_by_route":        arrivalsByRoute,
		"timestamp":                time.Now().Format(time.RFC3339),
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

	route, err := statisticsRouteFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	statsMutex.Lock()
	defer statsMutex.Unlock()

	routeCache := cachedAccuracyMap[route]
	if routeCache == nil {
		routeCache = map[string]*StationAccuracyResponse{}
	}

	// Return all stations from the route cache (includes stations with 0 arrivals).
	response := make([]StationAccuracyResponse, 0, len(routeCache))
	nonZeroCount := 0
	for _, stationData := range routeCache {
		response = append(response, *stationData)
		if stationData.InboundTotal > 0 || stationData.OutboundTotal > 0 {
			nonZeroCount++
		}
	}

	// Debug log to show what we're returning
	debugf("/api/statistics called for %s: Returning %d stations (%d with data)\n", route, len(response), nonZeroCount)
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
		pred, _, err := fetchPredictionSingle(routeGreenB, "70135", 0, "place-brico") // example stop ID (Packards Corner)
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
			predictionDataMap = make(map[string]map[string]map[int]map[string][]PredictionData)
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
			for _, stations := range stationAccuracyMap {
				for _, directions := range stations {
					for _, stats := range directions {
						totalPredictions += stats.Total
					}
				}
			}
			stationAccuracyMap = make(map[string]map[string]map[int]*StationStats)
			stationRecentDiffMap = make(map[string]map[string]map[int]*RecentPredictionDiff)
			cachedAccuracyMap = make(map[string]map[string]*StationAccuracyResponse)
			statsMutex.Unlock()
			// Re-seed all configured stations so /api/statistics never returns an empty station list after reset.
			initializeStationMaps()

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

	// Start goroutines to monitor actual train arrivals by route.
	go actualArrivalMoment(routeGreenB)
	go actualArrivalMoment(routeGreenC)
	go actualArrivalMoment(routeGreenD)
	go actualArrivalMoment(routeGreenE)
	fmt.Printf("Started monitoring %s, %s, %s and %s lines for arrivals...\n", routeGreenB, routeGreenC, routeGreenD, routeGreenE)
	// =============================LISTENING====================================
	// Start goroutine to constantly listen for arrivals
	go func() {
		fmt.Println("Started listening to ArrivalChannel...")
		// this is equivalent to <-Arrival Channel.
		for arrival := range ArrivalChannel {
			arrivalRoute, err := normalizeRoute(arrival.Route)
			if err != nil {
				debugf("Skipping arrival with unsupported route %q: %v\n", arrival.Route, err)
				continue
			}
			debugln("\n=== TRAIN ARRIVAL DETECTED ===")
			debugf("Route: %s\n", arrivalRoute)
			debugf("Station (Place ID): %s\n", arrival.StationPlaceID)
			debugf("Station (Stop ID): %s\n", arrival.StationStopID)
			debugf("Train ID: %s\n", arrival.TrainID)
			debugf("Direction: %d\n", arrival.Direction)
			debugf("Actual Arrival Time: %s\n", arrival.ArrivalTime.Format(time.RFC3339))
			debugln("==============================")

			// Compare with prediction data and score accuracy
			// Use StationPlaceID (parent station) to match how predictions are stored
			predictionMutex.RLock()
			debugf("Looking up predictions for Route %s, Place ID: %s, Direction: %d, Train: %s\n", arrivalRoute, arrival.StationPlaceID, arrival.Direction, arrival.TrainID)

			// Debug: Show what's in predictionDataMap for this station
			if stationPreds, hasStation := predictionDataMap[arrivalRoute][arrival.StationPlaceID]; hasStation {
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

			if predictions, exists := predictionDataMap[arrivalRoute][arrival.StationPlaceID][arrival.Direction][arrival.TrainID]; exists {
				debugf("Found %d prediction(s) for this train:\n", len(predictions))
				targetObservationTime := arrival.ArrivalTime.Add(-5 * time.Minute)
				gradedCount := 0
				correctCount := 0
				incorrectCount := 0
				middlePrediction, hasMiddlePrediction := selectMiddlePrediction(predictions, arrival.ArrivalTime)
				middleDiffMinutes := 0.0

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

				if hasMiddlePrediction && middlePrediction.ArrivalTime != nil {
					middleDiffMinutes = arrival.ArrivalTime.Sub(*middlePrediction.ArrivalTime).Minutes()
				}

				if gradedCount > 0 || hasMiddlePrediction {
					statsMutex.Lock()
					if stationAccuracyMap[arrivalRoute] == nil {
						stationAccuracyMap[arrivalRoute] = make(map[string]map[int]*StationStats)
					}
					if stationAccuracyMap[arrivalRoute][arrival.StationPlaceID] == nil {
						stationAccuracyMap[arrivalRoute][arrival.StationPlaceID] = make(map[int]*StationStats)
					}
					if stationAccuracyMap[arrivalRoute][arrival.StationPlaceID][arrival.Direction] == nil {
						stationAccuracyMap[arrivalRoute][arrival.StationPlaceID][arrival.Direction] = &StationStats{}
					}

					if gradedCount > 0 {
						// Update statistics for this arrival with all eligible predictions.
						stats := stationAccuracyMap[arrivalRoute][arrival.StationPlaceID][arrival.Direction]
						stats.Total += gradedCount
						stats.Correct += correctCount
						stats.Incorrect += incorrectCount
					}

					if hasMiddlePrediction && middlePrediction.ArrivalTime != nil {
						if stationRecentDiffMap[arrivalRoute] == nil {
							stationRecentDiffMap[arrivalRoute] = make(map[string]map[int]*RecentPredictionDiff)
						}
						if stationRecentDiffMap[arrivalRoute][arrival.StationPlaceID] == nil {
							stationRecentDiffMap[arrivalRoute][arrival.StationPlaceID] = make(map[int]*RecentPredictionDiff)
						}
						stationRecentDiffMap[arrivalRoute][arrival.StationPlaceID][arrival.Direction] = &RecentPredictionDiff{
							DifferenceMinutes:         middleDiffMinutes,
							ArrivalTime:               arrival.ArrivalTime,
							PredictionObservationTime: middlePrediction.ObservationTime,
							PredictionArrivalTime:     *middlePrediction.ArrivalTime,
							TrainID:                   arrival.TrainID,
						}
					}

					updateCachedAccuracy(arrivalRoute, arrival.StationPlaceID)

					if gradedCount > 0 {
						stats := stationAccuracyMap[arrivalRoute][arrival.StationPlaceID][arrival.Direction]
						accuracy := 0.0
						if stats.Total > 0 {
							accuracy = float64(stats.Correct) / float64(stats.Total) * 100
						}
						directionName := "Outbound"
						if arrival.Direction == 1 {
							directionName = "Inbound"
						}
						debugf("\nSTATION STATISTICS [%s / %s - %s]:\n", arrivalRoute, arrival.StationPlaceID, directionName)
						debugf("   Total Predictions: %d\n", stats.Total)
						debugf("   Correct (<=3 min): %d\n", stats.Correct)
						debugf("   Wrong (>3 min):   %d\n", stats.Incorrect)
						debugf("   Accuracy Rate:    %.2f%%\n", accuracy)
					}

					if hasMiddlePrediction && middlePrediction.ArrivalTime != nil {
						debugf("Most recent arrival diff saved [%s/%s dir=%d]: %.2f min (middle snapshot at %s)\n",
							arrivalRoute, arrival.StationPlaceID, arrival.Direction, middleDiffMinutes,
							middlePrediction.ObservationTime.Format(time.RFC3339))
					}
					statsMutex.Unlock()
				}

				if gradedCount == 0 {
					debugln("No predictions observed at least 5 minutes before arrival")
				}
				// Cleanup: Remove evaluated predictions to prevent memory leak
				predictionMutex.RUnlock()
				predictionMutex.Lock()
				delete(predictionDataMap[arrivalRoute][arrival.StationPlaceID][arrival.Direction], arrival.TrainID)
				if len(predictionDataMap[arrivalRoute][arrival.StationPlaceID][arrival.Direction]) == 0 {
					delete(predictionDataMap[arrivalRoute][arrival.StationPlaceID], arrival.Direction)
				}
				if len(predictionDataMap[arrivalRoute][arrival.StationPlaceID]) == 0 {
					delete(predictionDataMap[arrivalRoute], arrival.StationPlaceID)
				}
				if len(predictionDataMap[arrivalRoute]) == 0 {
					delete(predictionDataMap, arrivalRoute)
				}
				predictionMutex.Unlock()
				debugf("Cleaned up predictions for route %s vehicle %s at station %s\n", arrivalRoute, arrival.TrainID, arrival.StationPlaceID)
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

		platformToParentByRoute := map[string]map[string]string{
			routeGreenB: {
				"70106": "place-lake", "70107": "place-lake",
				"70110": "place-sougr", "70111": "place-sougr",
				"70112": "place-chill", "70113": "place-chill",
				"70114": "place-chswk", "70115": "place-chswk",
				"70116": "place-sthld", "70117": "place-sthld",
				"70120": "place-wascm", "70121": "place-wascm",
				"70124": "place-wrnst", "70125": "place-wrnst",
				"70126": "place-alsgr", "70127": "place-alsgr",
				"70128": "place-grigg", "70129": "place-grigg",
				"70130": "place-harvd", "70131": "place-harvd",
				"70134": "place-brico", "70135": "place-brico",
				"170136": "place-babck", "170137": "place-babck",
				"170140": "place-amory", "170141": "place-amory",
				"70144": "place-bucen", "70145": "place-bucen",
				"70146": "place-buest", "70147": "place-buest",
				"70148": "place-bland", "70149": "place-bland",
				"70150": "place-kencl", "70151": "place-kencl", "71150": "place-kencl", "71151": "place-kencl",
				"70152": "place-hymnl", "70153": "place-hymnl",
				"70154": "place-coecl", "70155": "place-coecl",
				"70156": "place-armnl", "70157": "place-armnl",
				"70158": "place-boyls", "70159": "place-boyls",
				"70196": "place-pktrm", "70197": "place-pktrm", "70198": "place-pktrm", "70199": "place-pktrm", "70200": "place-pktrm", "71199": "place-pktrm",
				"70201": "place-gover", "70202": "place-gover",
			},
			routeGreenC: {
				"70211": "place-smary", "70212": "place-smary",
				"70213": "place-hwsst", "70214": "place-hwsst",
				"70215": "place-kntst", "70216": "place-kntst",
				"70217": "place-stpul", "70218": "place-stpul",
				"70219": "place-cool", "70220": "place-cool",
				"70223": "place-sumav", "70224": "place-sumav",
				"70225": "place-bndhl", "70226": "place-bndhl",
				"70227": "place-fbkst", "70228": "place-fbkst",
				"70229": "place-bcnwa", "70230": "place-bcnwa",
				"70231": "place-tapst", "70232": "place-tapst",
				"70233": "place-denrd", "70234": "place-denrd",
				"70235": "place-engav", "70236": "place-engav",
				"70237": "place-clmnl", "70238": "place-clmnl",
				"70150": "place-kencl", "70151": "place-kencl", "71150": "place-kencl", "71151": "place-kencl",
				"70152": "place-hymnl", "70153": "place-hymnl",
				"70154": "place-coecl", "70155": "place-coecl",
				"70156": "place-armnl", "70157": "place-armnl",
				"70158": "place-boyls", "70159": "place-boyls",
				"70196": "place-pktrm", "70197": "place-pktrm", "70198": "place-pktrm", "70199": "place-pktrm", "70200": "place-pktrm", "71199": "place-pktrm",
				"70201": "place-gover", "70202": "place-gover",
			},
			routeGreenD: {
				"70160": "place-river", "70161": "place-river",
				"70162": "place-woodl", "70163": "place-woodl",
				"70164": "place-waban", "70165": "place-waban",
				"70166": "place-eliot", "70167": "place-eliot",
				"70168": "place-newtn", "70169": "place-newtn",
				"70170": "place-newto", "70171": "place-newto",
				"70172": "place-chhil", "70173": "place-chhil",
				"70174": "place-rsmnl", "70175": "place-rsmnl",
				"70176": "place-bcnfd", "70177": "place-bcnfd",
				"70178": "place-brkhl", "70179": "place-brkhl",
				"70180": "place-bvmnl", "70181": "place-bvmnl",
				"70182": "place-longw", "70183": "place-longw",
				"70186": "place-fenwy", "70187": "place-fenwy",
				"70150": "place-kencl", "70151": "place-kencl", "71150": "place-kencl", "71151": "place-kencl",
				"70152": "place-hymnl", "70153": "place-hymnl",
				"70154": "place-coecl", "70155": "place-coecl",
				"70156": "place-armnl", "70157": "place-armnl",
				"70158": "place-boyls", "70159": "place-boyls",
				"70196": "place-pktrm", "70197": "place-pktrm", "70198": "place-pktrm", "70199": "place-pktrm", "70200": "place-pktrm", "71199": "place-pktrm",
				"70201": "place-gover", "70202": "place-gover",
				"70203": "place-haecl", "70204": "place-haecl",
				"70205": "place-north", "70206": "place-north",
				"70207": "place-spmnl", "70208": "place-spmnl",
				"70209": "place-lech", "70210": "place-lech",
				// Union Square: use parent ID directly (GLX station, platform IDs resolved dynamically)
				"place-unsqu": "place-unsqu",
			},
			routeGreenE: {
				"70260": "place-hsmnl", "70261": "place-hsmnl",
				"70257": "place-bckhl", "70258": "place-bckhl",
				"70255": "place-rvrwy", "70256": "place-rvrwy",
				"70253": "place-mispk", "70254": "place-mispk",
				"70251": "place-fenwd", "70252": "place-fenwd",
				"70249": "place-brmnl", "70250": "place-brmnl",
				"70247": "place-lngmd", "70248": "place-lngmd",
				"70245": "place-mfa", "70246": "place-mfa",
				"70243": "place-nuniv", "70244": "place-nuniv",
				"70241": "place-symcl", "70242": "place-symcl",
				"70239": "place-prmnl", "70240": "place-prmnl",
				"70150": "place-kencl", "70151": "place-kencl", "71150": "place-kencl", "71151": "place-kencl",
				"70152": "place-hymnl", "70153": "place-hymnl",
				"70154": "place-coecl", "70155": "place-coecl",
				"70156": "place-armnl", "70157": "place-armnl",
				"70158": "place-boyls", "70159": "place-boyls",
				"70196": "place-pktrm", "70197": "place-pktrm", "70198": "place-pktrm", "70199": "place-pktrm", "70200": "place-pktrm", "71199": "place-pktrm",
				"70201": "place-gover", "70202": "place-gover",
				"70203": "place-haecl", "70204": "place-haecl",
				"70205": "place-north", "70206": "place-north",
				"70207": "place-spmnl", "70208": "place-spmnl",
				"70209": "place-lech", "70210": "place-lech",
				// GLX stations: use parent IDs directly (platform IDs resolved dynamically)
				"place-esomr": "place-esomr",
				"place-gilmn": "place-gilmn",
				"place-mgngl": "place-mgngl",
				"place-balsq": "place-balsq",
				"place-mdftf": "place-mdftf",
			},
		}

		// Helper function to fetch predictions for a platform
		fetchForPlatform := func(route, platformID, parentID string) {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Recovered from panic in fetchPredictionSingle for route %s platform %s: %v\n", route, platformID, r)
				}
			}()

			// Fetch both directions
			if _, _, err := fetchPredictionSingle(route, platformID, 0, parentID); err != nil {
				if err.Error() != "no predictions available" {
					fmt.Printf("Error fetching prediction for route %s platform %s direction 0: %v\n", route, platformID, err)
				}
			}
			if _, _, err := fetchPredictionSingle(route, platformID, 1, parentID); err != nil {
				if err.Error() != "no predictions available" {
					fmt.Printf("Error fetching prediction for route %s platform %s direction 1: %v\n", route, platformID, err)
				}
			}
		}

		// Fetch predictions immediately on startup
		debugln("Fetching initial predictions for all configured platforms...")
		for route, platformToParent := range platformToParentByRoute {
			for platformID, parentID := range platformToParent {
				go fetchForPlatform(route, platformID, parentID)
			}
		}

		// Then continue with periodic updates
		for {
			<-ticker.C
			debugln("Fetching updated predictions...")
			for route, platformToParent := range platformToParentByRoute {
				for platformID, parentID := range platformToParent {
					go fetchForPlatform(route, platformID, parentID)
				}
			}
		}
	}()

	// Block forever to keep the program running while goroutines work
	select {}
}

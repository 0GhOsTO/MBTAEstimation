package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

var key string

var predictionDataMap = make(map[string]map[int]map[string][]PredictionData)
var predictionMutex sync.RWMutex

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
	"70136":       "Babcock Street",
	"70138":       "Pleasant Street",
	"70140":       "Saint Paul Street",
	"70142":       "Boston University West",
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

type PredictionData struct {
	ObservationTime time.Time
	StopID          string
	ArrivalTime     *time.Time
	Status          string
	VehicleID       string
	TripID          string
}

type PredictionsResponse struct {
	Data []struct {
		Attributes struct {
			ArrivalTime *string `json:"arrival_time"`
			Status      *string `json:"status"`
		} `json:"attributes"`
		Relationships struct {
			Stop struct {
				Data struct {
					ID string `json:"id"`
				} `json:"data"`
			} `json:"stop"`
			Vehicle struct {
				Data struct {
					ID string `json:"id"`
				} `json:"data"`
			} `json:"vehicle"`
			Trip struct {
				Data struct {
					ID string `json:"id"`
				} `json:"data"`
			} `json:"trip"`
		} `json:"relationships"`
	} `json:"data"`
}

type StationAccuracyResponse struct {
	StationID        string  `json:"station_id"`
	StationName      string  `json:"station_name"`
	InboundAccuracy  float64 `json:"inbound_accuracy"`
	InboundTotal     int     `json:"inbound_total"`
	OutboundAccuracy float64 `json:"outbound_accuracy"`
	OutboundTotal    int     `json:"outbound_total"`
}

var statsMutex sync.Mutex
var allowedOrigins = parseAllowedOrigins()

func init() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("⚠️  No .env file found (this is normal in production)")
	}

	key = os.Getenv("MBTA_API_KEY")
	if key == "" {
		fmt.Println("⚠️ MBTA_API_KEY not set; network polling will be disabled until configured")
	}

	initDB()
}

func fetchPredictionsForDirection(ctx context.Context, direction int, platformToParent map[string]string) error {
	if db == nil {
		return fmt.Errorf("database not configured")
	}

	url := fmt.Sprintf("https://api-v3.mbta.com/predictions?filter[route]=Green-B&filter[direction_id]=%d&sort=arrival_time&page[limit]=500", direction)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-api-key", key)

	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("mbta predictions API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return err
	}

	var predResp PredictionsResponse
	if err := json.Unmarshal(body, &predResp); err != nil {
		return err
	}

	if len(predResp.Data) == 0 {
		return nil
	}

	observationTime := time.Now().UTC()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO prediction_snapshots
			(observed_at, station_id, direction, vehicle_id, trip_id, predicted_arrival, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (observed_at, station_id, direction, vehicle_id, predicted_arrival)
		DO NOTHING`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	inserted := 0
	for _, pred := range predResp.Data {
		stopID := pred.Relationships.Stop.Data.ID
		parentID, ok := platformToParent[stopID]
		if !ok {
			continue
		}

		vehicleID := pred.Relationships.Vehicle.Data.ID
		if vehicleID == "" {
			continue
		}

		var arrival *time.Time
		if pred.Attributes.ArrivalTime != nil && *pred.Attributes.ArrivalTime != "" {
			if parsed, parseErr := time.Parse(time.RFC3339, *pred.Attributes.ArrivalTime); parseErr == nil {
				p := parsed.UTC()
				arrival = &p
			}
		}

		var status string
		if pred.Attributes.Status != nil {
			status = *pred.Attributes.Status
		}

		if _, err := stmt.ExecContext(
			ctx,
			observationTime,
			normalizeStationKey(parentID),
			direction,
			vehicleID,
			pred.Relationships.Trip.Data.ID,
			arrival,
			status,
		); err != nil {
			return err
		}
		inserted++
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	fmt.Printf("✅ Stored %d Green-B prediction snapshots for direction %d\n", inserted, direction)
	return nil
}

func handleDebug(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}

	if err := applyCORSHeaders(w, r); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var predictionCount int
	var gradedCount int
	_ = db.QueryRow(`SELECT COUNT(*) FROM prediction_snapshots`).Scan(&predictionCount)
	_ = db.QueryRow(`SELECT COUNT(*) FROM prediction_snapshots WHERE graded = TRUE`).Scan(&gradedCount)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":            "running",
		"api_key_set":       key != "",
		"predictions_count": predictionCount,
		"graded_count":      gradedCount,
		"timestamp":         time.Now().Format(time.RFC3339),
	})
}

func handleGetStatistics(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}

	if err := applyCORSHeaders(w, r); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	statsMutex.Lock()
	defer statsMutex.Unlock()

	query := `
		SELECT
			station_id,
			direction,
			COUNT(*) AS total,
			AVG(CASE WHEN ABS(error_seconds) <= 180 THEN 1.0 ELSE 0.0 END) * 100 AS accuracy
		FROM prediction_snapshots
		WHERE graded = TRUE
		GROUP BY station_id, direction`

	rows, err := db.QueryContext(r.Context(), query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	responseByStation := make(map[string]*StationAccuracyResponse, len(greenBStationNames))
	for stationID, stationName := range greenBStationNames {
		responseByStation[stationID] = &StationAccuracyResponse{
			StationID:   stationID,
			StationName: stationName,
		}
	}

	for rows.Next() {
		var stationID string
		var direction int
		var total int
		var accuracy sql.NullFloat64
		if err := rows.Scan(&stationID, &direction, &total, &accuracy); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if responseByStation[stationID] == nil {
			responseByStation[stationID] = &StationAccuracyResponse{StationID: stationID, StationName: stationID}
		}

		if direction == 1 {
			responseByStation[stationID].InboundTotal = total
			if accuracy.Valid {
				responseByStation[stationID].InboundAccuracy = accuracy.Float64
			}
		} else {
			responseByStation[stationID].OutboundTotal = total
			if accuracy.Valid {
				responseByStation[stationID].OutboundAccuracy = accuracy.Float64
			}
		}
	}

	response := make([]StationAccuracyResponse, 0, len(responseByStation))
	for _, row := range responseByStation {
		row.InboundAccuracy = math.Round(row.InboundAccuracy*100) / 100
		row.OutboundAccuracy = math.Round(row.OutboundAccuracy*100) / 100
		response = append(response, *row)
	}

	_ = json.NewEncoder(w).Encode(response)
}

func parseAllowedOrigins() map[string]struct{} {
	raw := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS"))
	if raw == "" {
		return map[string]struct{}{"http://localhost:5173": {}}
	}

	origins := make(map[string]struct{})
	for _, part := range strings.Split(raw, ",") {
		origin := strings.TrimSpace(part)
		if origin != "" {
			origins[origin] = struct{}{}
		}
	}

	return origins
}

func applyCORSHeaders(w http.ResponseWriter, r *http.Request) error {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return nil
	}

	if _, ok := allowedOrigins[origin]; !ok {
		return errors.New("origin not allowed")
	}

	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Vary", "Origin")
	return nil
}

func gradeArrivalAgainstPredictions(arrival ArrivalInfo) {
	if db == nil {
		return
	}

	cutoff := arrival.ArrivalTime.Add(-5 * time.Minute)
	rows, err := db.Query(`
		SELECT id, predicted_arrival
		FROM prediction_snapshots
		WHERE station_id = $1
		  AND direction = $2
		  AND vehicle_id = $3
		  AND graded = FALSE
		  AND predicted_arrival IS NOT NULL
		  AND observed_at <= $4
		ORDER BY observed_at ASC`, arrival.StationPlaceID, arrival.Direction, arrival.TrainID, cutoff)
	if err != nil {
		fmt.Printf("⚠️ failed to query snapshots for grading: %v\n", err)
		return
	}
	defer rows.Close()

	gradedCount := 0
	for rows.Next() {
		var id int64
		var predictedArrival time.Time
		if err := rows.Scan(&id, &predictedArrival); err != nil {
			fmt.Printf("⚠️ failed to parse snapshot row: %v\n", err)
			continue
		}

		errorSeconds := arrival.ArrivalTime.Sub(predictedArrival).Seconds()
		_, err := db.Exec(`
			UPDATE prediction_snapshots
			SET graded = TRUE,
				actual_arrival = $2,
				error_seconds = $3
			WHERE id = $1`, id, arrival.ArrivalTime, errorSeconds)
		if err != nil {
			fmt.Printf("⚠️ failed to update graded snapshot: %v\n", err)
			continue
		}
		gradedCount++
	}

	if gradedCount > 0 {
		fmt.Printf("📊 Graded %d predictions for station=%s direction=%d vehicle=%s\n", gradedCount, arrival.StationPlaceID, arrival.Direction, arrival.TrainID)
	}
}

func main() {
	if key == "" {
		panic("❌ MBTA_API_KEY environment variable not set")
	}

	port := getEnvOrDefault("PORT", "8080")

	http.HandleFunc("/api/statistics", handleGetStatistics)
	http.HandleFunc("/api/v1/statistics", handleGetStatistics)
	http.HandleFunc("/api/debug", handleDebug)
	go func() {
		addr := "0.0.0.0:" + port
		fmt.Printf("🌐 HTTP API server starting on %s\n", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			panic(err)
		}
	}()

	go actualArrivalMoment("Green-B")
	go func() {
		for arrival := range ArrivalChannel {
			arrival.StationPlaceID = normalizeStationKey(arrival.StationPlaceID)
			gradeArrivalAgainstPredictions(arrival)
		}
	}()

	platformToParent := map[string]string{
		"70106": "place-lake", "70107": "place-lake", "70110": "place-sougr", "70111": "place-sougr",
		"70112": "place-chill", "70113": "place-chill", "70114": "place-chswk", "70115": "place-chswk",
		"70116": "place-sthld", "70117": "place-sthld", "70120": "place-wascm", "70121": "place-wascm",
		"70124": "place-wrnst", "70125": "place-wrnst", "70126": "place-alsgr", "70127": "place-alsgr",
		"70128": "place-grigg", "70129": "place-grigg", "70130": "place-harvd", "70131": "place-harvd",
		"70134": "place-brico", "70135": "place-brico", "70136": "70136", "70137": "70136", "170136": "70136", "170137": "70136",
		"70138": "70138", "70139": "70138", "70140": "70140", "70141": "70140", "170140": "70140", "170141": "70140",
		"70142": "70142", "70143": "70142", "70144": "place-bucen", "70145": "place-bucen", "70146": "place-buest", "70147": "place-buest",
		"70148": "place-bland", "70149": "place-bland", "70150": "place-kencl", "70151": "place-kencl", "71150": "place-kencl", "71151": "place-kencl",
		"70152": "place-hymnl", "70153": "place-hymnl", "70154": "place-coecl", "70155": "place-coecl", "70156": "place-armnl", "70157": "place-armnl",
		"70158": "place-boyls", "70159": "place-boyls", "70196": "place-pktrm", "70197": "place-pktrm", "70198": "place-pktrm",
		"70199": "place-pktrm", "70200": "place-pktrm", "71199": "place-pktrm", "70201": "place-gover", "70202": "place-gover",
	}

	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := fetchPredictionsForDirection(ctx, 0, platformToParent); err != nil {
				fmt.Printf("⚠️ outbound prediction fetch failed: %v\n", err)
			}
			if err := fetchPredictionsForDirection(ctx, 1, platformToParent); err != nil {
				fmt.Printf("⚠️ inbound prediction fetch failed: %v\n", err)
			}
			cancel()
			<-ticker.C
		}
	}()

	select {}
}

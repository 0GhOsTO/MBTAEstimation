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
	"strings"
	"sync"
	"time"
)

// save the actual Train Information
var actualTrainInfo = make(map[string]ActualData)

// save the next stop information for each vehicle
var vehicleNextStop = make(map[string]string)

// ArrivalInfo holds information about a train arrival
type ArrivalInfo struct {
	StationPlaceID string    // Station ID (e.g., "place-lake")
	StationStopID  string    // Stop ID (e.g., "70106")
	TrainID        string    // Vehicle/Train ID
	Direction      int       // Direction ID (0 or 1)
	ArrivalTime    time.Time // Time of arrival
}

// Channel to send arrival notifications
var ArrivalChannel = make(chan ArrivalInfo, 1000)

// dynamically store stop geolocations fetched from the API
var dynamicStopGeoLocation = make(map[string][2]float64)

// dynamically store stop ID to parent station (place ID) mapping
var stopToParentStation = make(map[string]string)

// track consecutive detections within 20m for each vehicle
var vehicleProximityCount = make(map[string]int)

// track if vehicle was within 20m on the last poll (for strict consecutive detection)
var vehicleLastPollWithin20m = make(map[string]bool)

var stationGeoLocation = map[string][2]float64{
	"70106": {42.34018, -71.16709}, // Boston College
	"70110": {42.33972, -71.16070}, // South Street
	"70112": {42.33843, -71.15274}, // Chestnut Hill Ave
	"70113": {42.33847, -71.15282}, // Chiswick Road
	"70114": {42.33907, -71.14606}, // Sutherland Road
	"70117": {42.34313, -71.14130}, // Washington Street
	"70121": {42.34413, -71.14262}, // Warren Street
	"70130": {42.35013, -71.13158}, // Allston Street
	"70134": {42.35118, -71.12192}, // Griggs Street
	"70144": {42.35005, -71.10722}, // Harvard Avenue
	"70146": {42.35103, -71.11671}, // Packards Corner
	"70153": {42.35188, -71.12068}, // Pleasant Street
	"70154": {42.34788, -71.08627}, // St. Paul Street
	"70155": {42.35018, -71.07710}, // Kent Street
	"70157": {42.34956, -71.09979}, // Blandford Street
	"70159": {42.34882, -71.09564}, // Kenmore
}

// Static mapping of platform IDs to parent station (place) IDs for Green Line
var staticStopToParentStation = map[string]string{
	// Green Line B - Boston College
	"70106": "place-lake",  // Boston College
	"70107": "place-lake",  // Boston College - Exit Only
	"70110": "place-sougr", // South Street
	"70111": "place-sougr", // South Street
	"70112": "place-chill", // Chestnut Hill Avenue
	"70113": "place-chill", // Chestnut Hill Avenue
	"70114": "place-chswk", // Chiswick Road
	"70115": "place-chswk", // Chiswick Road
	"70116": "place-sthld", // Sutherland Road
	"70117": "place-sthld", // Sutherland Road
	"70120": "place-wascm", // Washington Street
	"70121": "place-wascm", // Washington Street
	"70124": "place-wrnst", // Warren Street
	"70125": "place-wrnst", // Warren Street
	"70126": "place-alsgr", // Allston Street
	"70127": "place-alsgr", // Allston Street
	"70128": "place-grigg", // Griggs Street
	"70129": "place-grigg", // Griggs Street
	"70130": "place-harvd", // Harvard Avenue
	"70131": "place-harvd", // Harvard Avenue
	"70134": "place-brico", // Packards Corner
	"70135": "place-brico", // Packards Corner
	// Note: Stops 70136-70143 have no parent station in MBTA API
	// They map to themselves as they are orphaned platforms
	"70136": "70136",       // Babcock Street
	"70137": "70137",       // Babcock Street
	"70138": "70138",       // Pleasant Street
	"70139": "70139",       // Pleasant Street
	"70140": "70140",       // Saint Paul Street (B)
	"70141": "70141",       // Saint Paul Street (B)
	"70142": "70142",       // Boston University West
	"70143": "70143",       // Boston University West
	"70144": "place-bucen", // Boston University Central
	"70145": "place-bucen", // Boston University Central
	"70146": "place-buest", // Boston University East
	"70147": "place-buest", // Boston University East
	"70148": "place-bland", // Blandford Street
	"70149": "place-bland", // Blandford Street
	"70150": "place-kencl", // Kenmore
	"70151": "place-kencl", // Kenmore (C/D)
	"70152": "place-hymnl", // Hynes Convention Center
	"70153": "place-hymnl", // Hynes Convention Center
	"70154": "place-coecl", // Copley
	"70155": "place-coecl", // Copley
	"70156": "place-armnl", // Arlington
	"70157": "place-armnl", // Arlington
	"70158": "place-boyls", // Boylston
	"70159": "place-boyls", // Boylston
	"70160": "place-river", // Riverside
	"70161": "place-river", // Riverside - Exit Only
	"70162": "place-woodl", // Woodland
	"70163": "place-woodl", // Woodland
	"70164": "place-waban", // Waban
	"70165": "place-waban", // Waban
	"70166": "place-eliot", // Eliot
	"70167": "place-eliot", // Eliot
	"70168": "place-newtn", // Newton Highlands
	"70169": "place-newtn", // Newton Highlands
	"70170": "place-newto", // Newton Centre
	"70171": "place-newto", // Newton Centre
	"70172": "place-chhil", // Chestnut Hill (D)
	"70173": "place-chhil", // Chestnut Hill (D)
	"70174": "place-rsmnl", // Reservoir
	"70175": "place-rsmnl", // Reservoir
	"70176": "place-bcnfd", // Beaconsfield
	"70177": "place-bcnfd", // Beaconsfield
	"70178": "place-brkhl", // Brookline Hills
	"70179": "place-brkhl", // Brookline Hills
	"70180": "place-bvmnl", // Brookline Village
	"70181": "place-bvmnl", // Brookline Village
	"70182": "place-longw", // Longwood (D)
	"70183": "place-longw", // Longwood (D)
	"70186": "place-fenwy", // Fenway
	"70187": "place-fenwy", // Fenway
	"70196": "place-pktrm", // Park Street (B)
	"70197": "place-pktrm", // Park Street (C)
	"70198": "place-pktrm", // Park Street (D)
	"70199": "place-pktrm", // Park Street (E)
	"70200": "place-pktrm", // Park Street
	"70201": "place-gover", // Government Center
	"70202": "place-gover", // Government Center
	"70203": "place-haecl", // Haymarket
	"70204": "place-haecl", // Haymarket
	"70205": "place-north", // North Station
	"70206": "place-north", // North Station
	"70207": "place-spmnl", // Science Park/West End
	"70208": "place-spmnl", // Science Park/West End
	"70209": "70209",       // Lechmere - Exit Only (no parent in API)
	"70210": "70210",       // Lechmere (no parent in API)
	"70211": "place-smary", // Saint Marys Street
	"70212": "place-smary", // Saint Marys Street
	"70213": "place-hwsst", // Hawes Street
	"70214": "place-hwsst", // Hawes Street
	"70215": "place-kntst", // Kent Street
	"70216": "place-kntst", // Kent Street
	"70217": "place-stpul", // Saint Paul Street (C)
	"70218": "place-stpul", // Saint Paul Street (C)
	"70219": "place-cool",  // Coolidge Corner
	"70220": "place-cool",  // Coolidge Corner
	"70223": "place-sumav", // Summit Avenue
	"70224": "place-sumav", // Summit Avenue
	"70225": "place-bndhl", // Brandon Hall
	"70226": "place-bndhl", // Brandon Hall
	"70227": "place-fbkst", // Fairbanks Street
	"70228": "place-fbkst", // Fairbanks Street
	"70229": "place-bcnwa", // Washington Square
	"70230": "place-bcnwa", // Washington Square
	"70231": "place-tapst", // Tappan Street
	"70232": "place-tapst", // Tappan Street
	"70233": "place-denrd", // Dean Road
	"70234": "place-denrd", // Dean Road
	"70235": "place-engav", // Englewood Avenue
	"70236": "place-engav", // Englewood Avenue
	"70237": "place-clmnl", // Cleveland Circle - Exit Only
	"70238": "place-clmnl", // Cleveland Circle
	"70239": "place-prmnl", // Prudential
	"70240": "place-prmnl", // Prudential
	"70241": "place-symcl", // Symphony
	"70242": "place-symcl", // Symphony
	"70243": "place-nuniv", // Northeastern University
	"70244": "place-nuniv", // Northeastern University
	"70245": "place-mfa",   // Museum of Fine Arts
	"70246": "place-mfa",   // Museum of Fine Arts
	"70247": "place-lngmd", // Longwood Medical Area
	"70248": "place-lngmd", // Longwood Medical Area
	"70249": "place-brmnl", // Brigham Circle
	"70250": "place-brmnl", // Brigham Circle
	"70251": "place-fenwd", // Fenwood Road
	"70252": "place-fenwd", // Fenwood Road
	"70253": "place-mispk", // Mission Park
	"70254": "place-mispk", // Mission Park
	"70255": "place-rvrwy", // Riverway
	"70256": "place-rvrwy", // Riverway
	"70257": "place-bckhl", // Back of the Hill
	"70258": "place-bckhl", // Back of the Hill
	"71150": "place-kencl", // Kenmore
	"71151": "place-kencl", // Kenmore
	"71199": "place-pktrm", // Park Street - Drop-off Only
}

// List of unique parent station IDs (place-* IDs) from staticStopToParentStation
var parentStationIDs = []string{
	"place-alsgr", // Allston Street
	"place-armnl", // Arlington
	"place-bcnfd", // Beaconsfield
	"place-bcnwa", // Washington Square
	"place-bckhl", // Back of the Hill
	"place-bland", // Blandford Street
	"place-bndhl", // Brandon Hall
	"place-boyls", // Boylston
	"place-brico", // Packards Corner
	"place-brkhl", // Brookline Hills
	"place-brmnl", // Brigham Circle
	"place-bucen", // Boston University Central
	"place-buest", // Boston University East
	"place-bvmnl", // Brookline Village
	"place-chhil", // Chestnut Hill (D)
	"place-chill", // Chestnut Hill Avenue
	"place-chswk", // Chiswick Road
	"place-clmnl", // Cleveland Circle
	"place-coecl", // Copley
	"place-cool",  // Coolidge Corner
	"place-denrd", // Dean Road
	"place-eliot", // Eliot
	"place-engav", // Englewood Avenue
	"place-fbkst", // Fairbanks Street
	"place-fenwd", // Fenwood Road
	"place-fenwy", // Fenway
	"place-gover", // Government Center
	"place-grigg", // Griggs Street
	"place-haecl", // Haymarket
	"place-harvd", // Harvard Avenue
	"place-hwsst", // Hawes Street
	"place-hymnl", // Hynes Convention Center
	"place-kencl", // Kenmore
	"place-kntst", // Kent Street
	"place-lake",  // Boston College
	"place-lngmd", // Longwood Medical Area
	"place-longw", // Longwood (D)
	"place-mfa",   // Museum of Fine Arts
	"place-mispk", // Mission Park
	"place-newtn", // Newton Highlands
	"place-newto", // Newton Centre
	"place-north", // North Station
	"place-nuniv", // Northeastern University
	"place-pktrm", // Park Street
	"place-prmnl", // Prudential
	"place-river", // Riverside
	"place-rsmnl", // Reservoir
	"place-rvrwy", // Riverway
	"place-smary", // Saint Marys Street
	"place-sougr", // South Street
	"place-spmnl", // Science Park/West End
	"place-stpul", // Saint Paul Street (C)
	"place-sthld", // Sutherland Road
	"place-sumav", // Summit Avenue
	"place-symcl", // Symphony
	"place-tapst", // Tappan Street
	"place-waban", // Waban
	"place-wascm", // Washington Street
	"place-woodl", // Woodland
	"place-wrnst", // Warren Street
	"70209",
	"70210",
	"70136",
	"70137",
	"70138",
	"70139",
	"70140",
	"70141",
	"70142",
	"70143",
}

// NEED TO START FROM HERE =======================================

// mutex for protecting shared maps from race conditions
var mapMutex sync.RWMutex

// Note: 'key' variable is declared and initialized in receiveAPIcall.go

type ActualData struct {
	VehicleID       string
	ObservationTime time.Time
	Status          string
	RelatedStop     string
	Latitude        float64
	Longitude       float64
	DirectionID     int
}

// Struct to unmarshal MBTA vehicles API response
type VehiclesResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Attributes struct {
			CurrentStatus       string  `json:"current_status"`
			Latitude            float64 `json:"latitude"`
			Longitude           float64 `json:"longitude"`
			UpdatedAt           string  `json:"updated_at"`
			CurrentStopSequence *int    `json:"current_stop_sequence"`
			DirectionID         int     `json:"direction_id"`
		} `json:"attributes"`
		Relationships struct {
			Stop struct {
				Data *struct {
					ID string `json:"id"`
				} `json:"data"`
			} `json:"stop"`
			Trip struct {
				Data *struct {
					ID string `json:"id"`
				} `json:"data"`
			} `json:"trip"`
		} `json:"relationships"`
	} `json:"data"`
}

// Struct to unmarshal MBTA schedule/predictions API response for next stops
type TripIDResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Attributes struct {
			ArrivalTime   *string `json:"arrival_time"`
			DepartureTime *string `json:"departure_time"`
			StopSequence  int     `json:"stop_sequence"`
		} `json:"attributes"`
		Relationships struct {
			Stop struct {
				Data struct {
					ID string `json:"id"`
				} `json:"data"`
			} `json:"stop"`
		} `json:"relationships"`
	} `json:"data"`
}

// Struct to unmarshal MBTA stops API response for geolocation
type SR struct {
	Data struct {
		ID         string `json:"id"`
		Attributes struct {
			Name         string  `json:"name"`
			Latitude     float64 `json:"latitude"`
			Longitude    float64 `json:"longitude"`
			LocationType int     `json:"location_type"` // 0 = stop/platform, 1 = station/parent
		} `json:"attributes"`
		Relationships struct {
			ParentStation struct {
				Data *struct {
					ID string `json:"id"`
				} `json:"data"`
			} `json:"parent_station"`
		} `json:"relationships"`
	} `json:"data"`
}

// StopsListResponse for searching parent stations
type StopsListResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Attributes struct {
			Name         string `json:"name"`
			LocationType int    `json:"location_type"`
		} `json:"attributes"`
	} `json:"data"`
}

// haversineDistance calculates the distance in meters between two geographic coordinates
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371000 // Earth's radius in meters

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

// fetchStopGeolocation fetches stop coordinates from MBTA API if not already cached
func fetchStopGeolocation(stopID string, client *http.Client) ([2]float64, error) {
	mapMutex.RLock()
	if coords, exists := stationGeoLocation[stopID]; exists {
		mapMutex.RUnlock()
		return coords, nil
	}
	if coords, exists := dynamicStopGeoLocation[stopID]; exists {
		mapMutex.RUnlock()
		return coords, nil
	}
	mapMutex.RUnlock()

	url := fmt.Sprintf("https://api-v3.mbta.com/stops/%s", stopID)
	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return [2]float64{}, err
	}
	req.Header.Set("x-api-key", key)

	resp, err := client.Do(req)
	if err != nil {
		return [2]float64{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return [2]float64{}, err
	}

	var stopData SR
	if err := json.Unmarshal(body, &stopData); err != nil {
		return [2]float64{}, err
	}

	coords := [2]float64{stopData.Data.Attributes.Latitude, stopData.Data.Attributes.Longitude}

	// Extract parent station ID if available
	parentStationID := stopID // Default to stop ID if no parent
	if stopData.Data.Relationships.ParentStation.Data != nil {
		parentStationID = stopData.Data.Relationships.ParentStation.Data.ID
	}

	mapMutex.Lock()
	dynamicStopGeoLocation[stopID] = coords
	stopToParentStation[stopID] = parentStationID
	mapMutex.Unlock()

	fmt.Printf("Fetched geolocation for stop %s (parent: %s): [%.5f, %.5f]\n", stopID, parentStationID, coords[0], coords[1])
	return coords, nil
}

func actualArrivalMoment(routeName string) {
	// Every 30 seconds ...
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	// Create HTTP client once with timeout to prevent hanging
	client := &http.Client{Timeout: 10 * time.Second}
	// call once right away
	for {
		// @TODO: NEED to handle in go routine
		// 1. Fetch data from the MBTA API the line.
		url := fmt.Sprintf("https://api-v3.mbta.com/vehicles?filter[route]=%s", routeName)
		req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			<-ticker.C
			continue
		}
		// Set the API key in the header.
		req.Header.Set("x-api-key", key)
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error fetching vehicles: %v\n", err)
			<-ticker.C
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading response body: %v\n", err)
			resp.Body.Close()
			<-ticker.C
			continue
		}
		resp.Body.Close()

		// Successfully received data from the MBTA API.
		// =============================================
		// TEST PRINT
		fmt.Println("Received: ", string(body))

		// 1. Save the status, related stop, longitude, latitude inside the actualTrainInfo map.
		var vehiclesResp VehiclesResponse
		err = json.Unmarshal(body, &vehiclesResp)
		if err != nil {
			fmt.Printf("Error parsing JSON: %v\n", err)
			<-ticker.C
			continue
		}

		for _, vehicle := range vehiclesResp.Data {
			observationTime, err := time.Parse(time.RFC3339, vehicle.Attributes.UpdatedAt)
			if err != nil {
				observationTime = time.Now()
			}

			relatedStop := ""
			if vehicle.Relationships.Stop.Data != nil {
				relatedStop = vehicle.Relationships.Stop.Data.ID
			}

			directionID := vehicle.Attributes.DirectionID

			mapMutex.Lock()
			actualTrainInfo[vehicle.ID] = ActualData{
				VehicleID:       vehicle.ID,
				ObservationTime: observationTime,
				Status:          vehicle.Attributes.CurrentStatus,
				RelatedStop:     relatedStop,
				Latitude:        vehicle.Attributes.Latitude,
				Longitude:       vehicle.Attributes.Longitude,
				DirectionID:     directionID,
			}
			mapMutex.Unlock()
		}

		fmt.Printf("Updated %d vehicles in actualTrainInfo\n", len(vehiclesResp.Data))

		// CLEANING THE STALE RESPONSE.
		currentVehicleIDs := make(map[string]bool)
		for _, vehicle := range vehiclesResp.Data {
			currentVehicleIDs[vehicle.ID] = true
		}

		mapMutex.Lock()
		for vehicleID := range actualTrainInfo {
			if !currentVehicleIDs[vehicleID] {
				delete(actualTrainInfo, vehicleID)
				delete(vehicleNextStop, vehicleID)
				delete(vehicleProximityCount, vehicleID)
				delete(vehicleLastPollWithin20m, vehicleID)
				fmt.Printf("Cleaned up stale vehicle: %s\n", vehicleID)
			}
		}
		mapMutex.Unlock()

		// 2. Update where is the next stop for the each vehicle.
		// Only update if the vehicle has left the 20m radius of the current next stop
		for _, vehicle := range vehiclesResp.Data {
			currentStatus := vehicle.Attributes.CurrentStatus

			// GTFS-RT assumes IN_TRANSIT_TO if status is missing/empty
			if currentStatus == "" {
				currentStatus = "IN_TRANSIT_TO"
			}

			// Check if vehicle already has a next stop assigned
			mapMutex.RLock()
			existingNextStop, hasExistingNextStop := vehicleNextStop[vehicle.ID]
			mapMutex.RUnlock()

			// If vehicle has an existing next stop, check if it's still within 40m
			withinRadius := false
			if hasExistingNextStop {
				nextStopCoords, err := fetchStopGeolocation(existingNextStop, client)
				if err == nil {
					distance := haversineDistance(
						vehicle.Attributes.Latitude, vehicle.Attributes.Longitude,
						nextStopCoords[0], nextStopCoords[1],
					)
					withinRadius = distance <= 40.0
					if withinRadius {
						fmt.Printf("Vehicle %s still within 40m of next stop %s (%.2fm) - not updating\n",
							vehicle.ID, existingNextStop, distance)
					}
				}
			}

			// Only update next stop if vehicle doesn't have one OR has left the 40m radius
			if !hasExistingNextStop || !withinRadius {
				// Check if vehicle is in transit or incoming - status tells us the referenced stop is the next stop
				isApproaching := currentStatus == "IN_TRANSIT_TO" || currentStatus == "INCOMING_AT"

				if isApproaching {
					// The relationships.stop.data.id already is the next stop - no API call needed
					if vehicle.Relationships.Stop.Data != nil {
						mapMutex.Lock()
						vehicleNextStop[vehicle.ID] = vehicle.Relationships.Stop.Data.ID
						mapMutex.Unlock()
						fmt.Printf("Vehicle %s (status: %s) heading to: %s\n", vehicle.ID, currentStatus, vehicle.Relationships.Stop.Data.ID)
					} else {
						// No stop relationship - clear stale entry
						mapMutex.Lock()
						delete(vehicleNextStop, vehicle.ID)
						mapMutex.Unlock()
						fmt.Printf("Vehicle %s (status: %s) has no stop relationship - clearing next stop\n", vehicle.ID, currentStatus)
					}
				} else if currentStatus == "STOPPED_AT" {
					// Vehicle is stopped, need to find next stop using predictions
					if vehicle.Relationships.Trip.Data != nil {
						tripID := vehicle.Relationships.Trip.Data.ID
						currentStopSeq := 0
						if vehicle.Attributes.CurrentStopSequence != nil {
							currentStopSeq = *vehicle.Attributes.CurrentStopSequence
						}

						// Fetch predictions for this trip to find the next stop
						predURL := fmt.Sprintf("https://api-v3.mbta.com/predictions?filter[trip]=%s&sort=stop_sequence", tripID)
						predReq, err := http.NewRequestWithContext(context.Background(), "GET", predURL, nil)
						if err != nil {
							fmt.Printf("Error creating prediction request for vehicle %s: %v\n", vehicle.ID, err)
							continue
						}
						predReq.Header.Set("x-api-key", key)

						predResp, err := client.Do(predReq)
						if err != nil {
							fmt.Printf("Error fetching predictions for vehicle %s: %v\n", vehicle.ID, err)
							continue
						}

						predBody, err := io.ReadAll(predResp.Body)
						predResp.Body.Close()
						if err != nil {
							fmt.Printf("Error reading prediction response for vehicle %s: %v\n", vehicle.ID, err)
							continue
						}

						var tripData TripIDResponse
						err = json.Unmarshal(predBody, &tripData)
						if err != nil {
							fmt.Printf("Error parsing prediction JSON for vehicle %s: %v\n", vehicle.ID, err)
							continue
						}

						// Find the next stop (stop with sequence strictly > current sequence)
						nextStop := ""
						for _, pred := range tripData.Data {
							if pred.Attributes.StopSequence > currentStopSeq {
								nextStop = pred.Relationships.Stop.Data.ID
								break
							}
						}

						if nextStop != "" {
							mapMutex.Lock()
							vehicleNextStop[vehicle.ID] = nextStop
							mapMutex.Unlock()
							fmt.Printf("Vehicle %s (status: %s) next stop after current: %s\n", vehicle.ID, currentStatus, nextStop)
						} else {
							fmt.Printf("Vehicle %s (status: %s) could not determine next stop\n", vehicle.ID, currentStatus)
						}
					} else {
						fmt.Printf("Vehicle %s (status: %s) has no trip data\n", vehicle.ID, currentStatus)
					}
				} else {
					// Handle other statuses by treating them like IN_TRANSIT_TO if a stop relationship exists
					if vehicle.Relationships.Stop.Data != nil {
						mapMutex.Lock()
						vehicleNextStop[vehicle.ID] = vehicle.Relationships.Stop.Data.ID
						mapMutex.Unlock()
						fmt.Printf("Vehicle %s (status: %s - treating as approaching) heading to: %s\n", vehicle.ID, currentStatus, vehicle.Relationships.Stop.Data.ID)
					} else {
						fmt.Printf("Vehicle %s (status: %s) has no stop relationship\n", vehicle.ID, currentStatus)
					}
				}
			}
		}

		// 3. Continuously poll from the API and check how close the vehicle is to the next stop.
		// Check the stationGeolocation map for the next stop ID to get its latitude and longitude.
		// If it exists, compare it with the vehicle's current latitude and longitude stored in actualTrainInfo map.
		// If not, perform the api call to get the next stop information and save into the stationGeolocation map.
		// If the vehicle's current geolocation is under 20m radius of the next stop's geolocation in 2 times in a row,
		// consider the vehicle is about arrive at the next stop.
		// print [STOPID, VEHICLEID, TIMESTAMP]

		// Create a snapshot of the data to avoid holding locks during iteration
		mapMutex.RLock()
		vehicleSnapshot := make(map[string]ActualData)
		for k, v := range actualTrainInfo {
			vehicleSnapshot[k] = v
		}
		nextStopSnapshot := make(map[string]string)
		for k, v := range vehicleNextStop {
			nextStopSnapshot[k] = v
		}
		mapMutex.RUnlock()

		for vehicleID, actualData := range vehicleSnapshot {
			nextStopID, hasNextStop := nextStopSnapshot[vehicleID]
			if !hasNextStop {
				// No next stop - reset proximity count and last poll state
				mapMutex.Lock()
				delete(vehicleProximityCount, vehicleID)
				delete(vehicleLastPollWithin20m, vehicleID)
				mapMutex.Unlock()

				continue
			}

			// Fetch stop coordinates (from cache or API)
			nextStopCoords, err := fetchStopGeolocation(nextStopID, client)
			if err != nil {
				fmt.Printf("Warning: Could not fetch geolocation for stop %s: %v\n", nextStopID, err)
				continue
			}

			// Calculate distance to next stop
			distance := haversineDistance(
				actualData.Latitude, actualData.Longitude,
				nextStopCoords[0], nextStopCoords[1],
			)

			mapMutex.Lock()
			// Check if vehicle was within 20m on last poll
			lastPollWithin := vehicleLastPollWithin20m[vehicleID]
			currentPollWithin := distance <= 20.0

			if currentPollWithin {
				// Within 20m on current poll
				if lastPollWithin {
					// Within 20m on BOTH last poll and current poll - strict consecutive detection
					vehicleProximityCount[vehicleID]++

					if vehicleProximityCount[vehicleID] == 2 {
						// Detected 2 consecutive times within 20m - vehicle is arriving
						// WILL BE IN FORM OF RETURN
						// Get parent station (place ID) for output
						placeID := staticStopToParentStation[nextStopID]
						if placeID == "" {
							// Check dynamic mapping if not in static (already holding write lock)
							placeID = stopToParentStation[nextStopID]
						}
						if placeID == "" {
							placeID = nextStopID // Fallback to stop ID if parent not found
						}
						fmt.Printf("[%s, %s, %s, Direction: %d] - Vehicle arriving (distance: %.2fm)\n",
							placeID, vehicleID, actualData.ObservationTime.Format(time.RFC3339), actualData.DirectionID, distance)

						// Send arrival information to channel
						arrivalInfo := ArrivalInfo{
							StationPlaceID: placeID,
							StationStopID:  nextStopID,
							TrainID:        vehicleID,
							Direction:      actualData.DirectionID,
							ArrivalTime:    actualData.ObservationTime,
						}

						// Non-blocking send to avoid deadlock if no receiver
						select {
						case ArrivalChannel <- arrivalInfo:
							fmt.Printf("Sent arrival info to channel: %+v\n", arrivalInfo)
						default:
							fmt.Println("Warning: Arrival channel full, dropping message")
						}
					} else if vehicleProximityCount[vehicleID] > 2 {
						// Already reported arrival, keep count high to avoid duplicate reports
						// Count will reset when vehicle leaves 20m radius
					}
				} else {
					// First time within 20m (or just returned to 20m radius)
					vehicleProximityCount[vehicleID] = 1
				}
				// Update state for next poll
				vehicleLastPollWithin20m[vehicleID] = true
			} else {
				// Not within 20m - reset everything
				vehicleProximityCount[vehicleID] = 0
				vehicleLastPollWithin20m[vehicleID] = false
			}
			mapMutex.Unlock()

		}

		<-ticker.C
	}
}

// findParentStationByName searches for a parent station (location_type=1) with a matching name
func findParentStationByName(stopName string, client *http.Client) (string, error) {
	// Search for stations on Green Line with matching name
	url := fmt.Sprintf("https://api-v3.mbta.com/stops?filter[route]=Green-B,Green-C,Green-D,Green-E&filter[location_type]=1")
	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("x-api-key", key)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var stopsResponse StopsListResponse
	if err := json.Unmarshal(body, &stopsResponse); err != nil {
		return "", err
	}

	// Look for exact match or close match
	stopNameLower := strings.ToLower(stopName)
	for _, stop := range stopsResponse.Data {
		if strings.ToLower(stop.Attributes.Name) == stopNameLower {
			return stop.ID, nil
		}
	}

	return "not-found", nil
}

// verifyStaticStopMappings checks all staticStopToParentStation mappings against the MBTA API
func verifyStaticStopMappings() {
	fmt.Println("=== Verifying staticStopToParentStation mappings ===")
	fmt.Printf("Total stops to verify: %d\n\n", len(staticStopToParentStation))

	client := &http.Client{Timeout: 10 * time.Second}
	correctCount := 0
	incorrectCount := 0
	errorCount := 0

	for stopID, expectedParent := range staticStopToParentStation {
		// Make API call to get stop info
		url := fmt.Sprintf("https://api-v3.mbta.com/stops/%s", stopID)
		req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
		if err != nil {
			fmt.Printf("❌ Error creating request for stop %s: %v\n", stopID, err)
			errorCount++
			continue
		}
		req.Header.Set("x-api-key", key)

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("❌ Error fetching stop %s: %v\n", stopID, err)
			errorCount++
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			fmt.Printf("❌ Error reading response for stop %s: %v\n", stopID, err)
			errorCount++
			continue
		}

		var stopResponse SR
		if err := json.Unmarshal(body, &stopResponse); err != nil {
			fmt.Printf("❌ Error parsing JSON for stop %s: %v\n", stopID, err)
			errorCount++
			continue
		}

		// Get actual parent station from API
		var actualParent string
		locationType := stopResponse.Data.Attributes.LocationType
		stopName := stopResponse.Data.Attributes.Name

		if stopResponse.Data.Relationships.ParentStation.Data != nil {
			actualParent = stopResponse.Data.Relationships.ParentStation.Data.ID
		} else {
			// No parent station - stop maps to itself
			if locationType == 1 {
				// This is a parent station, so it should map to itself
				actualParent = stopID
			} else {
				// Orphaned platform - maps to itself
				fmt.Printf("⚠️  Stop %s ('%s') has no parent, mapping to itself\n", stopID, stopName)
				actualParent = stopID
			}
		}

		// Compare
		if actualParent == expectedParent {
			correctCount++
			fmt.Printf("✓ Stop %s: %s (correct, location_type=%d)\n", stopID, expectedParent, locationType)
		} else {
			incorrectCount++
			fmt.Printf("✗ Stop %s: Expected '%s', Got '%s' (MISMATCH, location_type=%d)\n", stopID, expectedParent, actualParent, locationType)
		}

		// Small delay to avoid rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("\n=== Verification Summary ===")
	fmt.Printf("✓ Correct: %d\n", correctCount)
	fmt.Printf("✗ Incorrect: %d\n", incorrectCount)
	fmt.Printf("❌ Errors: %d\n", errorCount)
	fmt.Printf("Total: %d\n", len(staticStopToParentStation))

	if incorrectCount > 0 {
		fmt.Println("\n⚠️  WARNING: Some mappings are incorrect!")
	} else if errorCount == 0 {
		fmt.Println("\n✓ All mappings verified successfully!")
	}
}

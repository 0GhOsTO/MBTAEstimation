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
	"sync"
	"time"
)

// save the actual Train Information
var actualTrainInfo = make(map[string]ActualData)

// save the next stop information for each vehicle
var vehicleNextStop = make(map[string]string)

// dynamically store stop geolocations fetched from the API
var dynamicStopGeoLocation = make(map[string][2]float64)

// track consecutive detections within 20m for each vehicle
var vehicleProximityCount = make(map[string]int)

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
	"70147": {42.35187, -71.12109}, // Babcock Street
	"70153": {42.35188, -71.12068}, // Pleasant Street
	"70154": {42.34788, -71.08627}, // St. Paul Street
	"70155": {42.35018, -71.07710}, // Kent Street
	"70157": {42.34956, -71.09979}, // Blandford Street
	"70159": {42.34882, -71.09564}, // Kenmore
}

// mutex for protecting shared maps from race conditions
var mapMutex sync.RWMutex

type ActualData struct {
	VehicleID       string
	ObservationTime time.Time
	Status          string
	RelatedStop     string
	Latitude        float64
	Longitude       float64
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
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
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
	mapMutex.Lock()
	dynamicStopGeoLocation[stopID] = coords
	mapMutex.Unlock()

	fmt.Printf("Fetched geolocation for stop %s: [%.5f, %.5f]\n", stopID, coords[0], coords[1])
	return coords, nil
}

func actualArrivalMoment(routeName string) {
	// Every 30 seconds ...
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	// call once right away
	for {
		// @TODO: NEED to handle in go routine
		// 1. Fetch data from the MBTA API the line.
		url := fmt.Sprintf("https://api-v3.mbta.com/vehicles?filter[route]=%s", routeName)
		req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
		if err != nil {
			panic(err)
		}
		// Set the API key in the header.
		req.Header.Set("x-api-key", key)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
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

			mapMutex.Lock()
			actualTrainInfo[vehicle.ID] = ActualData{
				VehicleID:       vehicle.ID,
				ObservationTime: observationTime,
				Status:          vehicle.Attributes.CurrentStatus,
				RelatedStop:     relatedStop,
				Latitude:        vehicle.Attributes.Latitude,
				Longitude:       vehicle.Attributes.Longitude,
			}
			mapMutex.Unlock()
		}

		fmt.Printf("Updated %d vehicles in actualTrainInfo\n", len(vehiclesResp.Data))

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

		for vehicleID, actualData := range actualTrainInfo {
			nextStopID, hasNextStop := vehicleNextStop[vehicleID]
			if !hasNextStop {
				// No next stop - reset proximity count
				mapMutex.Lock()
				delete(vehicleProximityCount, vehicleID)
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
			if distance <= 20.0 {
				// Within 20m - increment count
				vehicleProximityCount[vehicleID]++

				if vehicleProximityCount[vehicleID] >= 2 {
					// Detected 2 consecutive times within 20m - vehicle is arriving
					// WILL BE IN FORM OF RETURN
					fmt.Printf("[%s, %s, %s] - Vehicle arriving (distance: %.2fm, count: %d)\n",
						nextStopID, vehicleID, actualData.ObservationTime.Format(time.RFC3339), distance, vehicleProximityCount[vehicleID])
				}
			} else {
				// Not within 20m --> GPS spike.  - reset count
				vehicleProximityCount[vehicleID] = 0
			}
			mapMutex.Unlock()

		}

		<-ticker.C
	}
}

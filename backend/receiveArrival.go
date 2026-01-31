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
	"net/http"
	"sync"
	"time"
)

// save the actual Train Information
var actualTrainInfo = make(map[string]ActualData)

// save the next stop information for each vehicle
var vehicleNextStop = make(map[string]string)

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
		for _, vehicle := range vehiclesResp.Data {
			currentStatus := vehicle.Attributes.CurrentStatus

			// GTFS-RT assumes IN_TRANSIT_TO if status is missing/empty
			if currentStatus == "" {
				currentStatus = "IN_TRANSIT_TO"
			}

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
						// Clear stale entry on error
						mapMutex.Lock()
						delete(vehicleNextStop, vehicle.ID)
						mapMutex.Unlock()
						continue
					}
					predReq.Header.Set("x-api-key", key)

					predResp, err := client.Do(predReq)
					if err != nil {
						fmt.Printf("Error fetching predictions for vehicle %s: %v\n", vehicle.ID, err)
						// Clear stale entry on error
						mapMutex.Lock()
						delete(vehicleNextStop, vehicle.ID)
						mapMutex.Unlock()
						continue
					}

					predBody, err := io.ReadAll(predResp.Body)
					predResp.Body.Close()
					if err != nil {
						fmt.Printf("Error reading prediction response for vehicle %s: %v\n", vehicle.ID, err)
						// Clear stale entry on error
						mapMutex.Lock()
						delete(vehicleNextStop, vehicle.ID)
						mapMutex.Unlock()
						continue
					}

					var tripData TripIDResponse
					err = json.Unmarshal(predBody, &tripData)
					if err != nil {
						fmt.Printf("Error parsing prediction JSON for vehicle %s: %v\n", vehicle.ID, err)
						// Clear stale entry on error
						mapMutex.Lock()
						delete(vehicleNextStop, vehicle.ID)
						mapMutex.Unlock()
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
						// Could not determine next stop - clear stale entry
						mapMutex.Lock()
						delete(vehicleNextStop, vehicle.ID)
						mapMutex.Unlock()
						fmt.Printf("Vehicle %s (status: %s) could not determine next stop - clearing entry\n", vehicle.ID, currentStatus)
					}
				} else {
					// No trip data - clear stale entry
					mapMutex.Lock()
					delete(vehicleNextStop, vehicle.ID)
					mapMutex.Unlock()
					fmt.Printf("Vehicle %s (status: %s) has no trip data - clearing next stop\n", vehicle.ID, currentStatus)
				}
			} else {
				// Handle other statuses by treating them like IN_TRANSIT_TO if a stop relationship exists
				if vehicle.Relationships.Stop.Data != nil {
					mapMutex.Lock()
					vehicleNextStop[vehicle.ID] = vehicle.Relationships.Stop.Data.ID
					mapMutex.Unlock()
					fmt.Printf("Vehicle %s (status: %s - treating as approaching) heading to: %s\n", vehicle.ID, currentStatus, vehicle.Relationships.Stop.Data.ID)
				} else {
					// No stop relationship - clear stale entry
					mapMutex.Lock()
					delete(vehicleNextStop, vehicle.ID)
					mapMutex.Unlock()
					fmt.Printf("Vehicle %s (status: %s) has no stop relationship - clearing next stop\n", vehicle.ID, currentStatus)
				}
			}
		}

		// 3. Continuously poll from the API and check how close the vehicle is to the next stop.
		// Compare the geolocation of the next stop and the vehicle's current geolocation.\
		// Checking the geolocation of the each station

		<-ticker.C
	}
}

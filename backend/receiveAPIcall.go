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
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

// grab the API key.
var key string

// PREDICTION Hashmap for saving which train arrives where at what time.
var trainInfo = make(map[string][]PredictionData)

// Temporary storage for predictions before determining next stop
var tempTrainInfo = make(map[string][]PredictionData)

// Final storage for each train's next stop prediction
var trainNextStop = make(map[string][]PredictionData)

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

func fetchPrediction(stopID string) ([]PredictionData, error) {
	// constructing the request.
	url := fmt.Sprintf("https://api-v3.mbta.com/predictions?filter[stop]=%s", stopID)
	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", key)
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON response
	var predResp PredictionsResponse
	if err := json.Unmarshal(body, &predResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal predictions: %w", err)
	}

	// Extract the fields you need
	predictions := make([]PredictionData, 0, len(predResp.Data))
	observationTime := time.Now() // Current time as observation time

	for _, pred := range predResp.Data {
		data := PredictionData{
			ObservationTime: observationTime,
			StopID:          pred.Relationships.Stop.Data.ID,
			VehicleID:       pred.Relationships.Vehicle.Data.ID,
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

	return predictions, nil
}

func fetchPrediction_single(stopID string, direction int) ([]PredictionData, error) {
	// constructing the request.
	//url := fmt.Sprintf("https://api-v3.mbta.com/predictions?filter[stop]=%s&filter[direction_id]=%d", stopID, direction)
	url := fmt.Sprintf("https://api-v3.mbta.com/predictions?filter[stop]=%s&filter[direction_id]=%d&sort=arrival_time&page[limit]=1", stopID, direction)
	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", key)
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON response
	var predResp PredictionsResponse
	if err := json.Unmarshal(body, &predResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal predictions: %w", err)
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

	return predictions, nil
}

func main_singlePrediction() {
	// B line stop IDs:
	/* 70111 70113 70115 70117 70121 70125 70127 70129
	70131 70135 70137 70139 70141 70143 70145 70147 70149 70196 71151
	*/

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

		ids, err := getTrainInRoute("Green-B")
		if err != nil {
			panic(err)
		}
		fmt.Println("Train IDs: ", ids)
		fmt.Println("Count: ", len(ids))

		//=======TESTING==========
		predictions, err := fetchPrediction_single("70135", 0) // example stop ID
		if err != nil {
			panic(err)
		}

		// Print extracted data
		for _, pred := range predictions {
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
}

// This will aggregate the each extracted prediction data for the vehicle ID.
func aggregateVehiclePrediction() {
	// Grab the routes for the type 0.
	// Grab the trains with the vehicles in the Green-B line.
	predictions, err := fetchPrediction("70135") // example stop ID
	if err != nil {
		panic(err)
	}
	// Test printing out the body if it correctly received.
	// ==========================================================
	// Print the extracted data. (for testing)
	for _, pred := range predictions {
		fmt.Println("=== Prediction ===")
		fmt.Printf("Observation Time: %s\n", pred.ObservationTime.Format(time.RFC3339))
		fmt.Printf("Stop ID: %s\n", pred.StopID)
		fmt.Printf("Vehicle ID: %s\n", pred.VehicleID)
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

// This grabs the vehicle IDs in the specific route.
// routes -> route ID -> vehicles
func getTrainInRoute(routeName string) ([]string, error) {

	// Grab the routes for the type 0.
	// Grab the trains with the vehicles in the Green-B line.
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
	// Close the response body when it is done.
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	// Test printing out the body if it correctly received.
	fmt.Println(string(body) + "\n")

	// Structure to unMarshall the response.
	// var vehicleResp struct {} --> declare anonymous struct type.
	// Data --> field name | []struct {} --> slice of anonymous struct type.
	// `json:"id"` --> pure GO syntax to map the JSON key "id" to the field ID.
	var vehicleResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	// How unMarshalling works.
	// Get the raw data from the body.
	// Unmarshall into the struct and save at the adress of vehicleResp.
	if err := json.Unmarshal(body, &vehicleResp); err != nil {
		return nil, err
	}

	// ID extraction from the response.
	// Avoid pre-filling the slice with empty strings; start with length 0 and set capacity.
	ids := make([]string, 0, len(vehicleResp.Data))
	// for loop explanation:
	// _ --> ignoring the index | v --> value at that index in the slice.
	for _, v := range vehicleResp.Data {
		ids = append(ids, v.ID)
	}

	return ids, nil
}

type VehicleState struct {
	CurrentStopSequence int
	CurrentStopID       string
	CurrentStatus       string
}

func fetchVehicleStates(routeName string) (map[string]VehicleState, error) {
	url := fmt.Sprintf("https://api-v3.mbta.com/vehicles?filter[route]=%s", routeName)

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", key)

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Structure to unmarshal the vehicle response
	var vehicleResp struct {
		Data []struct {
			ID         string `json:"id"`
			Attributes struct {
				CurrentStopSequence *int    `json:"current_stop_sequence"`
				CurrentStatus       *string `json:"current_status"`
				DirectionID         *int    `json:"direction_id"`
			} `json:"attributes"`
			Relationships struct {
				Stop struct {
					Data *struct {
						ID   string `json:"id"`
						Type string `json:"type"`
					} `json:"data"`
				} `json:"stop"`
				Trip struct {
					Data *struct {
						ID   string `json:"id"`
						Type string `json:"type"`
					} `json:"data"`
				} `json:"trip"`
			} `json:"relationships"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &vehicleResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal vehicle response: %w", err)
	}

	// Build the vehicle state map
	vehicleStates := make(map[string]VehicleState)

	for _, vehicle := range vehicleResp.Data {
		state := VehicleState{}

		if vehicle.Attributes.CurrentStopSequence != nil {
			state.CurrentStopSequence = *vehicle.Attributes.CurrentStopSequence
		}

		if vehicle.Attributes.CurrentStatus != nil {
			state.CurrentStatus = *vehicle.Attributes.CurrentStatus
		}

		if vehicle.Relationships.Stop.Data != nil {
			state.CurrentStopID = vehicle.Relationships.Stop.Data.ID
		}

		vehicleStates[vehicle.ID] = state

		// ROBUST APPROACH: vehicle → trip → predictions → next stop
		// 1. Get the vehicle's trip_id
		if vehicle.Relationships.Trip.Data != nil {
			tripID := vehicle.Relationships.Trip.Data.ID
			currentStopSeq := 0
			if vehicle.Attributes.CurrentStopSequence != nil {
				currentStopSeq = *vehicle.Attributes.CurrentStopSequence
			}

			// 2. Call predictions with filter[trip] and sort by stop_sequence
			predictionURL := fmt.Sprintf("https://api-v3.mbta.com/predictions?filter[trip]=%s&sort=stop_sequence", tripID)
			predReq, err := http.NewRequestWithContext(context.Background(), "GET", predictionURL, nil)
			if err == nil {
				predReq.Header.Set("x-api-key", key)
				predResp, err := (&http.Client{}).Do(predReq)
				if err == nil {
					defer predResp.Body.Close()
					predBody, err := io.ReadAll(predResp.Body)
					if err == nil {
						// Unmarshal predictions response with stop_sequence
						var predResponse struct {
							Data []struct {
								ID         string `json:"id"`
								Attributes struct {
									ArrivalTime   *string `json:"arrival_time"`
									DepartureTime *string `json:"departure_time"`
									Status        *string `json:"status"`
									StopSequence  *int    `json:"stop_sequence"`
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

						if err := json.Unmarshal(predBody, &predResponse); err == nil {
							// 3. Process all predictions for this vehicle
							var nextPrediction *PredictionData
							var allPredictions []PredictionData
							observationTime := time.Now()

							for _, pred := range predResponse.Data {
								// Create prediction data for every stop in the trip
								data := PredictionData{
									ObservationTime: observationTime,
									StopID:          pred.Relationships.Stop.Data.ID,
									VehicleID:       vehicle.ID,
									TripID:          tripID,
								}

								if pred.Attributes.ArrivalTime != nil && *pred.Attributes.ArrivalTime != "" {
									if arrivalTime, err := time.Parse(time.RFC3339, *pred.Attributes.ArrivalTime); err == nil {
										data.ArrivalTime = &arrivalTime
									}
								}

								if pred.Attributes.DepartureTime != nil && *pred.Attributes.DepartureTime != "" {
									if departureTime, err := time.Parse(time.RFC3339, *pred.Attributes.DepartureTime); err == nil {
										data.DepartureTime = &departureTime
									}
								}

								if pred.Attributes.Status != nil {
									data.Status = *pred.Attributes.Status
								}

								// Add to all predictions for trainInfo
								allPredictions = append(allPredictions, data)

								// Find the next stop: smallest stop_sequence > current_stop_sequence
								if pred.Attributes.StopSequence != nil && *pred.Attributes.StopSequence > currentStopSeq {
									if nextPrediction == nil {
										// First valid next stop (since sorted by stop_sequence)
										temp := data
										nextPrediction = &temp
									}
								}
							}

							// 4. Save all predictions to trainInfo (full trip data)
							if len(allPredictions) > 0 {
								trainInfo[vehicle.ID] = allPredictions
							}

							// 5. Save only the next stop to trainNextStop
							if nextPrediction != nil {
								trainNextStop[vehicle.ID] = []PredictionData{*nextPrediction}
							}
						}
					}
				}
			}
		}
	}

	return vehicleStates, nil
}

func main() {
	// B line stop IDs:
	/* 70111 70113 70115 70117 70121 70125 70127 70129
	70131 70135 70137 70139 70141 70143 70145 70147 70149 70196 71151
	*/

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

		// Fetch vehicle states and populate both trainInfo and trainNextStop
		vs, error := fetchVehicleStates("Green-B")
		if error != nil {
			panic(error)
		}
		fmt.Println("\n\nVehicle States: ", vs)
		fmt.Println("Next stop predictions:", trainNextStop)
		// Clear tempTrainInfo for next iteration
		tempTrainInfo = make(map[string][]PredictionData)
	}
}

// Summarizing the progress for readme
// Had hard time to understand the API due to the error case when it returns blank or null
// Hard time calculating the arrival time for the returned data structure

//Mathematical equations
//error = | predicted_arrival_time - actual_arrival_time |

		fmt.Println("\n=== Vehicle States ===")
		for vehicleID, state := range vs {
			fmt.Printf("Vehicle %s: Stop %s, Sequence %d, Status %s\n", 
				vehicleID, state.CurrentStopID, state.CurrentStopSequence, state.CurrentStatus)
		}

		fmt.Println("\n=== Train Info (All Predictions per Vehicle) ===")
		for vehicleID, predictions := range trainInfo {
			fmt.Printf("Vehicle %s has %d predictions for its trip\n", vehicleID, len(predictions))
		}

		fmt.Println("\n=== Train Next Stop (Only Next Stop per Vehicle) ===")
		for vehicleID, predictions := range trainNextStop {
			if len(predictions) > 0 {
				pred := predictions[0]
				fmt.Printf("Vehicle %s → Next Stop: %s", vehicleID, pred.StopID)
				if pred.ArrivalTime != nil {
					fmt.Printf(" (ETA: %s)", pred.ArrivalTime.Format("15:04:05"))
				}
				fmt.Println()
			}
		}
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
	"time"

	"github.com/joho/godotenv"
)

// grab the API key.
var key string

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

func fetchPrediction(stopID string) {

	// constructing the request.
	url := fmt.Sprintf("https://api-v3.mbta.com/predictions?filter[stop]=%s", stopID)
	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("x-api-key", key)
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
}

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

		ids, err := getTrainInRoute("Green-B")
		if err != nil {
			panic(err)
		}
		fmt.Println("Train IDs: ", ids)
		fmt.Println("Count: ", len(ids))

		//=======TESTING==========
		// fetchPrediction("70135") // example stop ID
	}
}

// Summarizing the progress for readme
// Had hard time to understand the API due to the error case when it returns blank or null
// Hard time calculating the arrival time for the returned data structure

//Mathematical equations
//error = | predicted_arrival_time - actual_arrival_time |

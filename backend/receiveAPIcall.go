// Requirement before running: create go module
// cd f:\CS\ProjectGithub\MBTAEstimation\backend; go mod init mbta-backend
// Create .env file in backend folder with content:
// MBTA_API_KEY=your_actual_api_key_here
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func fetchPrediction(stopID string) {

	// grab the API key.
	key := os.Getenv("MBTA_API_KEY")
	if key == "" {
		panic("MBTA_API_KEY not set")
	}

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

func genTrainTable(trainID string) {

}

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("No .env found")
	}

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
		fetchPrediction("70135") // example stop ID
	}
}

// Summarizing the progress for readme
// Had hard time to understand the API due to the error case when it returns blank or null
// Hard time calculating the arrival time for the returned data structure

//Mathematical equations
//error = | predicted_arrival_time - actual_arrival_time |

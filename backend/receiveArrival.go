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

func actualArrivalMoment(routeName string) {
	// 1. Every 30 seconds ...
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	// call once right away
	for {
		<-ticker.C
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
		// Close the response body when it is done.
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		// Test printing out the body if it correctly received.
		fmt.Println(string(body) + "\n")
	}
}

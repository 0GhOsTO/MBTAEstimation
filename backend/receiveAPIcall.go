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

	"github.com/joho/godotenv"
)

func fetchPrediction(stopID string) {
	key := os.Getenv("MBTA_API_KEY")
	if key == "" {
		panic("MBTA_API_KEY not set")
	}
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

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("No .env found")
	}
	fetchPrediction("70020") // example stop ID
}

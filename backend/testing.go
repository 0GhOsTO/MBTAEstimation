package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

var apiKey string

func init() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("No .env found")
	}
	apiKey = os.Getenv("MBTA_API_KEY")
	if apiKey == "" {
		panic("MBTA_API_KEY not set")
	}
}

// Struct to unmarshal stop response
type StopResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Attributes struct {
			Name      string  `json:"name"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"attributes"`
	} `json:"data"`
}

// Fetch all stops for a specific route
func fetchRouteStops(routeID string) ([]StopResponse, error) {
	url := fmt.Sprintf("https://api-v3.mbta.com/stops?filter[route]=%s", routeID)
	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", apiKey)

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

	var stopResp StopResponse
	if err := json.Unmarshal(body, &stopResp); err != nil {
		return nil, err
	}

	return []StopResponse{stopResp}, nil
}

// Print geolocations for Green-B line stops
func printGreenBStops() {
	fmt.Println("Fetching Green-B line stops...")

	stopResponses, err := fetchRouteStops("Green-B")
	if err != nil {
		fmt.Printf("Error fetching stops: %v\n", err)
		return
	}

	if len(stopResponses) == 0 {
		fmt.Println("No stops found")
		return
	}

	stopResp := stopResponses[0]
	fmt.Printf("\nFound %d stops on Green-B line:\n\n", len(stopResp.Data))

	for _, stop := range stopResp.Data {
		fmt.Printf("%-6s  %s\n", stop.ID, stop.Attributes.Name)
		fmt.Printf("        %.5f, %.5f\n\n", stop.Attributes.Latitude, stop.Attributes.Longitude)
	}
}

// Print geolocations in map format for easy copying
func printGreenBStopsAsMap() {
	fmt.Println("Fetching Green-B line stops for map format...")

	stopResponses, err := fetchRouteStops("Green-B")
	if err != nil {
		fmt.Printf("Error fetching stops: %v\n", err)
		return
	}

	if len(stopResponses) == 0 {
		fmt.Println("No stops found")
		return
	}

	stopResp := stopResponses[0]
	fmt.Printf("\nvar stationGeoLocation = map[string][2]float64{\n")

	for _, stop := range stopResp.Data {
		fmt.Printf("\t\"%s\": {%.5f, %.5f},  // %s\n",
			stop.ID,
			stop.Attributes.Latitude,
			stop.Attributes.Longitude,
			stop.Attributes.Name)
	}

	fmt.Printf("}\n")
}

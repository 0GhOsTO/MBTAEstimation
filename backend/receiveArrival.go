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
	TripID         string    // Trip ID for better matching reliability
}

// Channel to send arrival notifications
var ArrivalChannel = make(chan ArrivalInfo, 5000)

// dynamically store stop geolocations fetched from the API
var dynamicStopGeoLocation = make(map[string][2]float64)

// dynamically store stop ID to parent station (place ID) mapping
var stopToParentStation = make(map[string]string)

// Canonical aliases so MBTA alternate IDs map to the same Green-B station keys used by frontend/backend stats.
var canonicalStationAliases = map[string]string{
	"place-babck": "70136", // Babcock Street
	"170136":      "70136", // Babcock Street inbound platform variant
	"170137":      "70136", // Babcock Street outbound platform variant
	"place-amory": "70140", // Amory Street
	"170140":      "70140", // Amory Street inbound platform variant
	"170141":      "70140", // Amory Street outbound platform variant
}

func normalizeStationKey(id string) string {
	if canonical, ok := canonicalStationAliases[id]; ok {
		return canonical
	}
	return id
}

// track which vehicle-stop pairs have already sent arrival notifications
var vehicleStopArrivalSent = make(map[string]bool)

var stationGeoLocation = map[string][2]float64{
	// Green Line B - Boston College Branch
	"70106": {42.340149, -71.167029}, // Boston College
	"70107": {42.340240, -71.166849}, // Boston College - Exit Only
	"70110": {42.339371, -71.157057}, // South Street
	"70111": {42.339581, -71.157499}, // South Street
	"70112": {42.338730, -71.152526}, // Chestnut Hill Avenue
	"70113": {42.338290, -71.153025}, // Chestnut Hill Avenue
	"70114": {42.340808, -71.150633}, // Chiswick Road
	"70115": {42.340540, -71.151140}, // Chiswick Road
	"70116": {42.341589, -71.146089}, // Sutherland Road
	"70117": {42.341577, -71.146607}, // Sutherland Road
	"70120": {42.344329, -71.142385}, // Washington Street
	"70121": {42.343974, -71.142731}, // Washington Street
	"70124": {42.348285, -71.140436}, // Warren Street
	"70125": {42.348819, -71.140051}, // Warren Street
	"70126": {42.348649, -71.137881}, // Allston Street
	"70127": {42.349251, -71.137398}, // Allston Street
	"70128": {42.348747, -71.134500}, // Griggs Street
	"70129": {42.348919, -71.134305}, // Griggs Street
	"70130": {42.350263, -71.131298}, // Harvard Avenue
	"70131": {42.350602, -71.130727}, // Harvard Avenue
	"70134": {42.351891, -71.125067}, // Packard's Corner
	"70135": {42.352136, -71.125126}, // Packard's Corner
	"70144": {42.350013, -71.106902}, // Boston University Central
	"70145": {42.349293, -71.106865}, // Boston University Central
	"70146": {42.349659, -71.103989}, // Boston University East
	"70147": {42.349148, -71.103767}, // Boston University East
	"70148": {42.348881, -71.100258}, // Blandford Street
	"70149": {42.349293, -71.100258}, // Blandford Street
	"70150": {42.348949, -71.095169}, // Kenmore
	"70151": {42.348949, -71.095169}, // Kenmore (C/D)
	"70152": {42.347888, -71.087903}, // Hynes Convention Center
	"70153": {42.347888, -71.087903}, // Hynes Convention Center
	"70154": {42.349871, -71.078049}, // Copley
	"70155": {42.350126, -71.077376}, // Copley
	"70156": {42.351635, -71.070694}, // Arlington
	"70157": {42.351902, -71.070893}, // Arlington
	"70158": {42.352816, -71.064262}, // Boylston
	"70159": {42.353214, -71.064545}, // Boylston
	// Green Line D - Riverside Branch
	"70160": {42.337317, -71.252256}, // Riverside
	"70161": {42.337348, -71.252236}, // Riverside
	"70162": {42.332703, -71.243055}, // Woodland
	"70163": {42.333094, -71.243659}, // Woodland
	"70164": {42.325695, -71.230476}, // Waban
	"70165": {42.325967, -71.230714}, // Waban
	"70166": {42.318871, -71.216420}, // Eliot
	"70167": {42.319214, -71.216949}, // Eliot
	"70168": {42.322738, -71.205082}, // Newton Highlands
	"70169": {42.322530, -71.205421}, // Newton Highlands
	"70170": {42.329552, -71.192024}, // Newton Centre
	"70171": {42.329400, -71.192622}, // Newton Centre
	"70172": {42.326799, -71.164146}, // Chestnut Hill (D)
	"70173": {42.326782, -71.164780}, // Chestnut Hill (D)
	"70174": {42.335181, -71.147879}, // Reservoir
	"70175": {42.335163, -71.148601}, // Reservoir
	"70176": {42.335860, -71.141426}, // Beaconsfield
	"70177": {42.335850, -71.140823}, // Beaconsfield
	"70178": {42.331470, -71.126999}, // Brookline Hills
	"70179": {42.331577, -71.127155}, // Brookline Hills
	"70180": {42.332614, -71.116751}, // Brookline Village
	"70181": {42.332570, -71.117041}, // Brookline Village
	"70182": {42.341808, -71.109777}, // Longwood (D)
	"70183": {42.341571, -71.110147}, // Longwood (D)
	"70186": {42.345328, -71.104269}, // Fenway
	"70187": {42.345029, -71.104968}, // Fenway
	// Green Line - Park Street Hub
	"70196": {42.356395, -71.062424}, // Park Street (B)
	"70197": {42.356395, -71.062424}, // Park Street (C)
	"70198": {42.356395, -71.062424}, // Park Street (D)
	"70199": {42.356395, -71.062424}, // Park Street (E)
	"70200": {42.356395, -71.062424}, // Park Street
	"71199": {42.356395, -71.062424}, // Park Street - Drop-off Only
	// Green Line - Downtown/North
	"70201": {42.359705, -71.059215}, // Government Center
	"70202": {42.359705, -71.059215}, // Government Center
	"70203": {42.363021, -71.058290}, // Haymarket
	"70204": {42.363021, -71.058290}, // Haymarket
	"70205": {42.365280, -71.060205}, // North Station
	"70206": {42.365280, -71.060205}, // North Station
	"70207": {42.366664, -71.067666}, // Science Park/West End
	"70208": {42.366664, -71.067666}, // Science Park/West End
	// Green Line C - Cleveland Circle Branch
	"70211": {42.345884, -71.107697}, // Saint Mary's Street
	"70212": {42.346007, -71.107166}, // Saint Mary's Street
	"70213": {42.344758, -71.111761}, // Hawes Street
	"70214": {42.344867, -71.111157}, // Hawes Street
	"70215": {42.344117, -71.114097}, // Kent Street
	"70216": {42.343927, -71.114569}, // Kent Street
	"70217": {42.343340, -71.116927}, // Saint Paul Street (C)
	"70218": {42.343118, -71.117498}, // Saint Paul Street (C)
	"70219": {42.342274, -71.120915}, // Coolidge Corner
	"70220": {42.342028, -71.121685}, // Coolidge Corner
	"70223": {42.341120, -71.125652}, // Summit Avenue
	"70224": {42.341027, -71.125759}, // Summit Avenue
	"70225": {42.339700, -71.129082}, // Brandon Hall
	"70226": {42.339623, -71.129192}, // Brandon Hall
	"70227": {42.339690, -71.131228}, // Fairbanks Street
	"70228": {42.339644, -71.131078}, // Fairbanks Street
	"70229": {42.339668, -71.135040}, // Washington Square
	"70230": {42.339461, -71.135649}, // Washington Square
	"70231": {42.338498, -71.138731}, // Tappan Street
	"70232": {42.338567, -71.138190}, // Tappan Street
	"70233": {42.337807, -71.141753}, // Dean Road
	"70234": {42.337628, -71.142309}, // Dean Road
	"70235": {42.336964, -71.145867}, // Englewood Avenue
	"70236": {42.337011, -71.145368}, // Englewood Avenue
	"70237": {42.336216, -71.149201}, // Cleveland Circle - Exit Only
	"70238": {42.336252, -71.148774}, // Cleveland Circle
	// Green Line E - Heath Street Branch
	"70239": {42.345570, -71.081696}, // Prudential
	"70240": {42.345570, -71.081696}, // Prudential
	"70241": {42.342687, -71.085056}, // Symphony
	"70242": {42.342687, -71.085056}, // Symphony
	"70243": {42.339897, -71.090210}, // Northeastern University
	"70244": {42.340222, -71.089200}, // Northeastern University
	"70245": {42.337875, -71.095240}, // Museum of Fine Arts
	"70246": {42.338017, -71.094682}, // Museum of Fine Arts
	"70247": {42.336080, -71.099883}, // Longwood Medical Area
	"70248": {42.336217, -71.099328}, // Longwood Medical Area
	"70249": {42.334229, -71.104609}, // Brigham Circle
	"70250": {42.334291, -71.104122}, // Brigham Circle
	"70251": {42.333740, -71.105721}, // Fenwood Road
	"70252": {42.333706, -71.105583}, // Fenwood Road
	"70253": {42.333279, -71.109276}, // Mission Park
	"70254": {42.333092, -71.109680}, // Mission Park
	"70255": {42.331391, -71.111925}, // Riverway
	"70256": {42.331871, -71.111961}, // Riverway
	"70257": {42.330139, -71.111313}, // Back of the Hill
	"70258": {42.330528, -71.111565}, // Back of the Hill
	// Kenmore additional platforms
	"71150": {42.348949, -71.095169}, // Kenmore
	"71151": {42.348949, -71.095169}, // Kenmore
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
	// Note: Stops 70136-70143 have no parent station in MBTA API.
	// Canonicalize each orphan pair to a single station key so both directions aggregate together.
	"70136": "70136",       // Babcock Street
	"70137": "70136",       // Babcock Street
	"70140": "70140",       // Amory Street (legacy IDs)
	"70141": "70140",       // Amory Street (legacy IDs)
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
	"70140",
	"70141",
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
	TripID          string
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
	parentStationID = normalizeStationKey(parentStationID)

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

		// 1. Save the status, related stop, longitude, latitude inside the actualTrainInfo map.
		var vehiclesResp VehiclesResponse
		err = json.Unmarshal(body, &vehiclesResp)
		if err != nil {
			fmt.Printf("Error parsing JSON: %v\n", err)
			<-ticker.C
			continue
		}

		// for the each vehicle, update the actualTrainInfo map with the latest status.
		for _, vehicle := range vehiclesResp.Data {
			// Skip vehicles with missing/empty IDs to avoid corrupting maps
			if vehicle.ID == "" {
				fmt.Println("Skipping vehicle with empty ID in actualTrainInfo update")
				continue
			}
			observationTime, err := time.Parse(time.RFC3339, vehicle.Attributes.UpdatedAt)
			if err != nil {
				observationTime = time.Now()
			}

			relatedStop := ""
			if vehicle.Relationships.Stop.Data != nil {
				relatedStop = vehicle.Relationships.Stop.Data.ID
			}

			tripID := ""
			if vehicle.Relationships.Trip.Data != nil {
				tripID = vehicle.Relationships.Trip.Data.ID
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
				TripID:          tripID,
			}
			mapMutex.Unlock()
		}

		fmt.Printf("Updated %d vehicles in actualTrainInfo\n", len(vehiclesResp.Data))

		// CLEANING THE STALE RESPONSE.
		currentVehicleIDs := make(map[string]bool)
		for _, vehicle := range vehiclesResp.Data {
			if vehicle.ID == "" {
				continue
			}
			currentVehicleIDs[vehicle.ID] = true
		}

		mapMutex.Lock()
		staleVehicles := make([]string, 0)
		for vehicleID := range actualTrainInfo {
			if !currentVehicleIDs[vehicleID] {
				delete(actualTrainInfo, vehicleID)
				delete(vehicleNextStop, vehicleID)
				// Clean up arrival tracking for this vehicle
				for key := range vehicleStopArrivalSent {
					if strings.HasPrefix(key, vehicleID+"-") {
						delete(vehicleStopArrivalSent, key)
					}
				}
				staleVehicles = append(staleVehicles, vehicleID)
				fmt.Printf("Cleaned up stale vehicle: %s\n", vehicleID)
			}
		}
		mapMutex.Unlock()

		// Also clean up prediction data for stale vehicles
		if len(staleVehicles) > 0 {
			predictionMutex.Lock()
			for _, vehicleID := range staleVehicles {
				for stopID := range predictionDataMap {
					for direction := range predictionDataMap[stopID] {
						if _, exists := predictionDataMap[stopID][direction][vehicleID]; exists {
							delete(predictionDataMap[stopID][direction], vehicleID)
							fmt.Printf("Cleaned up predictions for stale vehicle %s at stop %s direction %d\n", vehicleID, stopID, direction)
						}
					}
				}
			}
			predictionMutex.Unlock()
		}

		// 2. Update where is the next stop for the each vehicle.
		// Only update if the vehicle has left the 20m radius of the current next stop
		for _, vehicle := range vehiclesResp.Data {
			// Skip vehicles with missing/empty IDs
			if vehicle.ID == "" {
				fmt.Println("Skipping vehicle with empty ID in next-stop update")
				continue
			}
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

			// Only update next stop if vehicle doesn't have one OR has left the 40m radius OR is stopped
			// Exception: STOPPED_AT vehicles should always update to find the next-next stop
			if !hasExistingNextStop || !withinRadius || currentStatus == "STOPPED_AT" {
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

						// Get current stop ID if available
						currentStopID := ""
						if vehicle.Relationships.Stop.Data != nil {
							currentStopID = vehicle.Relationships.Stop.Data.ID
						}

						// Try to determine current stop sequence
						currentStopSeq := -1
						if vehicle.Attributes.CurrentStopSequence != nil {
							currentStopSeq = *vehicle.Attributes.CurrentStopSequence
						} else if currentStopID != "" {
							// Infer sequence from stop ID by searching tripData
							for _, pred := range tripData.Data {
								if pred.Relationships.Stop.Data.ID == currentStopID {
									currentStopSeq = pred.Attributes.StopSequence
									fmt.Printf("Vehicle %s: Inferred stop_sequence=%d from stop_id=%s\n", vehicle.ID, currentStopSeq, currentStopID)
									break
								}
							}
						}

						if currentStopSeq < 0 {
							fmt.Printf("Vehicle %s STOPPED_AT but can't determine stop_sequence; skipping update\n", vehicle.ID)
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
		// If the vehicle's current geolocation is under 20m radius of the next stop's geolocation,
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
			// Priority: If the train is STOPPED_AT, treat its current RelatedStop as arrived
			// regardless of what nextStopID says (nextStopID can be wrong).
			if actualData.Status == "STOPPED_AT" && actualData.RelatedStop != "" {
				var (
					shouldSendArrival bool
					arrivalInfo       ArrivalInfo
					arrivalKeyToMark  string
				)

				mapMutex.Lock()
				arrivalKey := fmt.Sprintf("%s-%s-%s", vehicleID, actualData.TripID, actualData.RelatedStop)
				alreadySent := vehicleStopArrivalSent[arrivalKey]

				if !alreadySent {
					placeID := staticStopToParentStation[actualData.RelatedStop]
					if placeID == "" {
						placeID = stopToParentStation[actualData.RelatedStop]
					}
					if placeID == "" {
						placeID = actualData.RelatedStop
					}
					placeID = normalizeStationKey(placeID)

					arrivalInfo = ArrivalInfo{
						StationPlaceID: placeID,
						StationStopID:  actualData.RelatedStop,
						TrainID:        vehicleID,
						Direction:      actualData.DirectionID,
						ArrivalTime:    actualData.ObservationTime,
						TripID:         actualData.TripID,
					}
					arrivalKeyToMark = arrivalKey
					shouldSendArrival = true
				}
				mapMutex.Unlock()

				if shouldSendArrival {
					fmt.Printf("[STOPPED_AT %s, %s, %s, Direction: %d] - Vehicle arrived at stop %s\n",
						arrivalInfo.StationPlaceID,
						arrivalInfo.TrainID,
						arrivalInfo.ArrivalTime.Format(time.RFC3339),
						arrivalInfo.Direction,
						arrivalInfo.StationStopID,
					)

					// Non-blocking send to avoid deadlock if no receiver
					sent := false
					select {
					case ArrivalChannel <- arrivalInfo:
						fmt.Printf("Sent arrival info to channel (STOPPED_AT): %+v\n", arrivalInfo)
						sent = true
					default:
						fmt.Println("Warning: Arrival channel full, dropping STOPPED_AT message")
					}

					if sent && arrivalKeyToMark != "" {
						mapMutex.Lock()
						vehicleStopArrivalSent[arrivalKeyToMark] = true
						mapMutex.Unlock()
					}
				} else {
					// If already sent and vehicle has moved far from the RelatedStop, clear tracking
					// This allows future arrivals at different stops to be detected
					if actualData.RelatedStop != "" {
						stopCoords, err := fetchStopGeolocation(actualData.RelatedStop, client)
						if err == nil {
							distance := haversineDistance(
								actualData.Latitude, actualData.Longitude,
								stopCoords[0], stopCoords[1],
							)
							if distance > 40.0 {
								mapMutex.Lock()
								for key := range vehicleStopArrivalSent {
									if strings.HasPrefix(key, vehicleID+"-") {
										delete(vehicleStopArrivalSent, key)
									}
								}
								mapMutex.Unlock()
							}
						}
					}
				}

				// STOPPED_AT vehicles don't use distance-based detection; skip to next vehicle
				continue
			}

			nextStopID, hasNextStop := nextStopSnapshot[vehicleID]
			if !hasNextStop {
				// No next stop for this vehicle
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

			// Decide what to do under lock, but perform slow work (logging, channel send)
			// outside the critical section.
			var (
				shouldSendArrival   bool
				arrivalInfo         ArrivalInfo
				arrivalKeyToMark    string
				shouldClearArrivals bool
			)

			mapMutex.Lock()
			currentPollWithin := distance <= 20.0

			if currentPollWithin {
				// Train is within 20m of the next stop - consider it arrived
				arrivalKey := fmt.Sprintf("%s-%s-%s", vehicleID, actualData.TripID, nextStopID)
				alreadySent := vehicleStopArrivalSent[arrivalKey]

				if !alreadySent {
					// Get parent station (place ID) for output
					placeID := staticStopToParentStation[nextStopID]
					if placeID == "" {
						// Check dynamic mapping if not in static
						placeID = stopToParentStation[nextStopID]
					}
					if placeID == "" {
						placeID = nextStopID // Fallback to stop ID if parent not found
					}
					placeID = normalizeStationKey(placeID)

					arrivalInfo = ArrivalInfo{
						StationPlaceID: placeID,
						StationStopID:  nextStopID,
						TrainID:        vehicleID,
						Direction:      actualData.DirectionID,
						ArrivalTime:    actualData.ObservationTime,
						TripID:         actualData.TripID,
					}
					arrivalKeyToMark = arrivalKey
					shouldSendArrival = true
				}
			} else {
				// If the vehicle is far away, clear its arrival tracking to allow future detections
				if distance > 40.0 {
					shouldClearArrivals = true
				}
			}
			mapMutex.Unlock()

			// Perform logging and channel send outside of the lock
			if shouldSendArrival {
				fmt.Printf("[%s, %s, %s, Direction: %d] - Vehicle arriving (distance: %.2fm)\n",
					arrivalInfo.StationPlaceID,
					arrivalInfo.TrainID,
					arrivalInfo.ArrivalTime.Format(time.RFC3339),
					arrivalInfo.Direction,
					distance,
				)

				// Non-blocking send to avoid deadlock if no receiver
				sent := false
				select {
				case ArrivalChannel <- arrivalInfo:
					fmt.Printf("Sent arrival info to channel: %+v\n", arrivalInfo)
					sent = true
				default:
					fmt.Println("Warning: Arrival channel full, dropping message")
				}

				// Only mark as sent if we actually enqueued the event
				if sent && arrivalKeyToMark != "" {
					mapMutex.Lock()
					vehicleStopArrivalSent[arrivalKeyToMark] = true
					mapMutex.Unlock()
				}
			}

			// If the vehicle is far away, clear its arrival tracking outside the main critical section
			if shouldClearArrivals {
				mapMutex.Lock()
				for key := range vehicleStopArrivalSent {
					if strings.HasPrefix(key, vehicleID+"-") {
						delete(vehicleStopArrivalSent, key)
					}
				}
				mapMutex.Unlock()
			}
		}

		<-ticker.C
	}
}

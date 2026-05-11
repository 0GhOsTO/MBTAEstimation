# [MBTA Estimation](https://mbta-estimation.vercel.app/)

A real-time MBTA Green Line prediction accuracy tracker with an interactive web interface.

## Project Overview

This project tracks and displays the trustworthiness of MBTA (Massachusetts Bay Transportation Authority) train arrival predictions by comparing predicted vs. actual arrival times.

## Project Origin

Being students at Boston University, the MBTA's green line is a significant mode of transportation, allowing for convenient travel and flexibility in planning trips. However, the stations running through the heart of Commonwealth Ave are notorious for delays (from traffic, construction, weather etc.), where students can be left stranded at stops waiting for a train that was supposed to arrive minutes ago. A problem especially during the winter months, where the temperatures can get uncomfortably cold. The MBTA posts timetables for scheduled arrivals for different stops, but sometimes that isn't enough. Our web app addresses that problem. We propose a metric called **trustworthiness score** to quantify how accurate predictions for train arrivals really are, and better inform our peers who frequent the B-Line.

Here: [Reddit discussion about this issue & conversation about our website](https://www.reddit.com/r/BostonU/comments/1rlqm5g/i_got_tired_of_waiting_for_the_green_line_so_i/)

#### Check-point
<img width="826" height="157" alt="image" src="https://github.com/user-attachments/assets/952e280e-1486-4cbf-96b8-68c852d726c2" />

## What is Trustworthiness Score?

Given a station $x$:

$$
Trustworthiness(x) = \frac{\sum(\text{correct arrival events})}{\text{num. arrival events graded}}
$$
  
For each train arrival event, we select exactly one prediction snapshot:

$$
\text{prediction snapshot time} \approx \text{actual arrival time} - 5 \text{ minutes}
$$

where a correct prediction is:

$$
\begin{cases}
1 & \text{if } |\text{predicted arrival time} - \text{actual arrival time}| \leq 3 \text{ min} \\
0 & \text{otherwise}
\end{cases}
$$

Only the predictions from the last two hours are stored and aggregated to calculated our score.

## Current Progress
### [Google Document of Progress](https://docs.google.com/document/d/1L1Hdq-_mwZ33vqSe75HmYA38mVIf2xTeh4whlfj5yr4/edit?usp=sharing)
### Frontend ✅
- **Interactive Map**: Leaflet-based map showing all Green Line stations (B, C, D, E)
- **Line Selection**: Dropdown menu to switch between Green Line branches
- **Real-time Display**: Shows trustworthiness percentage for selected stations
- **LaTeX Equations**: Mathematical formulas displayed with KaTeX rendering
- **Collapsible Info**: Dropdown button to show/hide calculation methodology
- **Activity Log**: Tracks user interactions and station selections
- **Responsive Design**: Modern glass-morphism UI with dark theme

### Backend 🚧 In Progress
- **Go-based API Client**: Connects to MBTA V3 API for real-time data
- **Vehicle Tracking**: Monitors all vehicles on Green Line routes
- **Prediction Collection**: Fetches and stores prediction data every 30 seconds
- **PostgreSQL Database**: Stores historical prediction and accuracy data
- **REST API**: Serves statistics to frontend with real-time accuracy metrics
- **Data Structures**:
  - `trainInfo`: Stores all predictions for each vehicle's entire trip
  - `trainNextStop`: Tracks only the next stop prediction for each vehicle
  - `vehicleStates`: Current vehicle status, stop, and sequence information
- **API Integration**:
  - Prediction API: Retrieves arrival/departure time predictions
  - Vehicle API: Gets current vehicle positions and status
  - Trip-based predictions: Finds next stops using trip IDs and stop sequences
- **Current Work**: 🔧 Implementing SQL queries and data aggregation for historical accuracy graphs
- **Hardships**:
  - API only provides the result sorted by station
  - Hard to find the train's next stop directly. 
  - API has multiple edges cases such as null/empty responses or the reality and API differing.
  - Scalability challenges such as limiting API calls and database writes

### Next Steps
- [ ] Complete SQL backend for accuracy graph visualization
- [ ] Time-series graphs showing accuracy trends over time
- [ ] Database optimization for historical data queries
- [ ] Real-time WebSocket updates to frontend
- [ ] Deploy on AWS
- [ ] Expand to other green line branches
- [ ] Provide score interpretations on the front end

## Screenshots

<img src="https://github.com/0GhOsTO/MBTAEstimation/blob/main/MBTAweb.png">

### Data Flow Visualization

<img src="https://github.com/0GhOsTO/MBTAEstimation/blob/main/dataVisualizationMBTA.gif">

This visualization demonstrates the real-time data flow of our prediction accuracy system. The **green dot** represents the train moving along the route, while the **blue dots** mark the station stops. Each stop concurrently receives predictions for upcoming train arrivals. When the train arrives at a stop, the system immediately "grades" the prediction accuracy by comparing the predicted arrival time against the actual arrival time, measuring how on-time the prediction was.

The grading uses the prediction observed closest to **5 minutes before** the actual arrival time (not the last-second prediction right before arrival).

## Tech Stack

**Frontend:**
- React + TypeScript
- Vite
- Leaflet (Interactive Maps)
- KaTeX (LaTeX Math Rendering)
- CSS3 (Glass-morphism design)

**Backend:**
- Go
- PostgreSQL
- MBTA V3 API
- godotenv (Environment variables)

## Setup

### Frontend
```bash
cd frontend
npm install
npm run dev
```

### Backend
```bash
cd backend
go mod init mbta-backend
go mod tidy
# Create .env file with:
# MBTA_API_KEY=your_green_b_api_key_here
# MBTA_API_KEY_C=your_green_c_api_key_here
# MBTA_API_KEY_D=your_green_d_api_key_here
# MBTA_API_KEY_E=your_green_e_api_key_here
go run receiveAPIcall.go
```

## API Documentation
- [MBTA V3 API Documentation](https://api-v3.mbta.com/docs/swagger/index.html)

## Notes
- Backend currently polls MBTA API every 30 seconds
- Handles edge cases like null/empty API responses
- Uses robust vehicle → trip → predictions flow for accuracy

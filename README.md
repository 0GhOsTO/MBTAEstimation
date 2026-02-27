# [MBTA Estimation](https://mbta-estimation.vercel.app/)

A real-time MBTA Green Line prediction accuracy tracker with an interactive web interface.

## Project Overview

This project tracks and displays the trustworthiness of MBTA (Massachusetts Bay Transportation Authority) train arrival predictions by comparing predicted vs. actual arrival times.

## Project Origin

Being students at Boston University, the MBTA's green line is a significant mode of transportation, allowing for convenient travel and flexibility in planning trips. However, the stations running through the heart of Commonwealth Ave are notorious for delays (from traffic, construction, weather etc.), where students can be left stranded at stops waiting for a train that was supposed to arrive minutes ago. A problem especially during the winter months, where the temperatures can get uncomfortably cold. The MBTA posts timetables for scheduled arrivals for different stops, but sometimes that isn't enough. Our web app addresses that problem. We propose a metric called **trustworthiness score** to quantify how accurate predictions for train arrivals really are, and better inform our peers who frequent the B-Line.

## What is Trustworthiness Score?

Given a station $x$:

$$
Trustworthiness(x) = \frac{\sum(\text{correct predictions})}{\text(num. predictions)}
$$
  
where a correct prediction is:

$$
\begin{cases}
1 & \text{if } |\text{predicted arrival time} - \text{actual arrival time}| \leq 3 \text{ min} \\
0 & \text{otherwise}
\end{cases}
$$

## Current Progress
### [Google Document of Progress](https://docs.google.com/document/d/1L1Hdq-_mwZ33vqSe75HmYA38mVIf2xTeh4whlfj5yr4/edit?usp=sharing)
### Frontend ✅
- **Interactive Map**: Leaflet-based map showing all Green Line stations (B, C, D, E)
- **Line Selection**: Dropdown menu to switch between Green Line branches
- **Real-time Display**: Shows trustworthiness percentage for selected stations
- **Activity Log**: Tracks user interactions and station selections
- **Responsive Design**: Modern glass-morphism UI with dark theme

### Backend 🚧 In Progress
- **Go-based API Client**: Connects to MBTA V3 API for real-time data
- **Vehicle Tracking**: Monitors all vehicles on Green Line routes
- **Prediction Collection**: Fetches and stores prediction data every 30 seconds
- **Data Structures**:
  - `trainInfo`: Stores all predictions for each vehicle's entire trip
  - `trainNextStop`: Tracks only the next stop prediction for each vehicle
  - `vehicleStates`: Current vehicle status, stop, and sequence information
- **API Integration**:
  - Prediction API: Retrieves arrival/departure time predictions
  - Vehicle API: Gets current vehicle positions and status
  - Trip-based predictions: Finds next stops using trip IDs and stop sequences
- **Hardships**:
  - API only provides the result sorted by station
  - Hard to find the train's next stop directly. 
  - API has multiple edges cases such as null/empty responses or the reality and API differing.
  - Scalability challenges such as limiting API calls and database writes

### Next Steps
- [ ] Database integration for persistent storage
- [ ] Real-time WebSocket updates to frontend
- [ ] Expand to other green line branches
- [ ] Provide score interpretations on the front end

## Screenshots

<img src="https://github.com/0GhOsTO/MBTAEstimation/blob/main/MBTAweb.png">

## Tech Stack

**Frontend:**
- React + TypeScript
- Vite
- Leaflet (Interactive Maps)
- CSS3 (Glass-morphism design)

**Backend:**
- Go
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
# Create .env file with: MBTA_API_KEY=your_key_here
go run receiveAPIcall.go
```

## API Documentation
- [MBTA V3 API Documentation](https://api-v3.mbta.com/docs/swagger/index.html)

## Notes
- Backend currently polls MBTA API every 30 seconds
- Handles edge cases like null/empty API responses
- Uses robust vehicle → trip → predictions flow for accuracy

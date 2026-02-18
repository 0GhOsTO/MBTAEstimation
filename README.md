# [MBTA Estimation](https://mbta-estimation.vercel.app/)

A real-time MBTA Green Line prediction accuracy tracker with an interactive web interface.

## Project Overview

This project tracks and displays the trustworthiness of MBTA (Massachusetts Bay Transportation Authority) train arrival predictions by comparing predicted vs. actual arrival times.

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
  - API has multilpe edges cases such as null/empty responses or the reality and API differing.

### Next Steps
- [ ] Calculate prediction accuracy (error = |predicted_time - actual_time|)
- [ ] Store historical prediction data
- [ ] Compute trustworthiness scores per station
- [ ] REST API endpoints to serve data to frontend
- [ ] Database integration for persistent storage
- [ ] Real-time WebSocket updates to frontend

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

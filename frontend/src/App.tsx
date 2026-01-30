import { useState } from 'react'
import { MapContainer, TileLayer, CircleMarker, Popup } from 'react-leaflet'
import 'leaflet/dist/leaflet.css'
import './App.css'

interface LogEntry {
  id: number;
  timestamp: string;
  message: string;
}

// Green Line stations with actual GPS coordinates
const stationsByLine: { [key: string]: Array<{ name: string; lat: number; lng: number }> } = {
  'Green-B': [
    { name: "Boston College", lat: 42.3396, lng: -71.1686 },
    { name: "South Street", lat: 42.3399, lng: -71.1571 },
    { name: "Chestnut Hill Ave", lat: 42.3387, lng: -71.1527 },
    { name: "Chiswick Road", lat: 42.3406, lng: -71.1504 },
    { name: "Sutherland Road", lat: 42.3410, lng: -71.1464 },
    { name: "Griggs Street", lat: 42.3481, lng: -71.1345 },
    { name: "Allston Street", lat: 42.3484, lng: -71.1373 },
    { name: "Warren Street", lat: 42.3485, lng: -71.1401 },
    { name: "Washington Street", lat: 42.3431, lng: -71.1420 },
    { name: "Babcock Street", lat: 42.3513, lng: -71.1218 },
    { name: "Pleasant Street", lat: 42.3513, lng: -71.1187 },
    { name: "Saint Paul Street", lat: 42.3511, lng: -71.1157 },
    { name: "BU West", lat: 42.3499, lng: -71.1138 },
    { name: "BU Central", lat: 42.3497, lng: -71.1070 },
    { name: "Blandford Street", lat: 42.3493, lng: -71.1002 },
    { name: "Kenmore", lat: 42.3488, lng: -71.0952 },
    { name: "Hynes Convention Center", lat: 42.3472, lng: -71.0876 },
    { name: "Copley", lat: 42.3499, lng: -71.0778 },
    { name: "Arlington", lat: 42.3524, lng: -71.0704 },
    { name: "Boylston", lat: 42.3530, lng: -71.0646 },
    { name: "Park Street", lat: 42.3563, lng: -71.0622 },
  ],
  'Green-C': [
    { name: "Cleveland Circle", lat: 42.3362, lng: -71.1495 },
    { name: "Englewood Ave", lat: 42.3375, lng: -71.1463 },
    { name: "Dean Road", lat: 42.3380, lng: -71.1417 },
    { name: "Tappan Street", lat: 42.3383, lng: -71.1387 },
    { name: "Washington Square", lat: 42.3433, lng: -71.1353 },
    { name: "Fairbanks Street", lat: 42.3476, lng: -71.1314 },
    { name: "Brandon Hall", lat: 42.3490, lng: -71.1291 },
    { name: "Summit Avenue", lat: 42.3502, lng: -71.1251 },
    { name: "Coolidge Corner", lat: 42.3420, lng: -71.1211 },
    { name: "Saint Paul Street", lat: 42.3511, lng: -71.1157 },
    { name: "Kenmore", lat: 42.3488, lng: -71.0952 },
    { name: "Hynes Convention Center", lat: 42.3472, lng: -71.0876 },
    { name: "Copley", lat: 42.3499, lng: -71.0778 },
    { name: "Arlington", lat: 42.3524, lng: -71.0704 },
    { name: "Boylston", lat: 42.3530, lng: -71.0646 },
    { name: "Park Street", lat: 42.3563, lng: -71.0622 },
  ],
  'Green-D': [
    { name: "Riverside", lat: 42.3367, lng: -71.2514 },
    { name: "Woodland", lat: 42.3332, lng: -71.2443 },
    { name: "Waban", lat: 42.3258, lng: -71.2308 },
    { name: "Eliot", lat: 42.3196, lng: -71.2164 },
    { name: "Newton Highlands", lat: 42.3217, lng: -71.2061 },
    { name: "Newton Centre", lat: 42.3290, lng: -71.1925 },
    { name: "Chestnut Hill", lat: 42.3266, lng: -71.1652 },
    { name: "Reservoir", lat: 42.3352, lng: -71.1496 },
    { name: "Beaconsfield", lat: 42.3358, lng: -71.1403 },
    { name: "Brookline Hills", lat: 42.3313, lng: -71.1267 },
    { name: "Brookline Village", lat: 42.3328, lng: -71.1163 },
    { name: "Longwood", lat: 42.3417, lng: -71.1099 },
    { name: "Fenway", lat: 42.3451, lng: -71.1041 },
    { name: "Kenmore", lat: 42.3488, lng: -71.0952 },
    { name: "Hynes Convention Center", lat: 42.3472, lng: -71.0876 },
    { name: "Copley", lat: 42.3499, lng: -71.0778 },
    { name: "Arlington", lat: 42.3524, lng: -71.0704 },
    { name: "Boylston", lat: 42.3530, lng: -71.0646 },
    { name: "Park Street", lat: 42.3563, lng: -71.0622 },
  ],
  'Green-E': [
    { name: "Heath Street", lat: 42.3282, lng: -71.1101 },
    { name: "Back of the Hill", lat: 42.3301, lng: -71.1104 },
    { name: "Riverway", lat: 42.3316, lng: -71.1107 },
    { name: "Mission Park", lat: 42.3338, lng: -71.1098 },
    { name: "Fenwood Road", lat: 42.3333, lng: -71.1056 },
    { name: "Brigham Circle", lat: 42.3342, lng: -71.1047 },
    { name: "Longwood Medical Area", lat: 42.3359, lng: -71.1005 },
    { name: "Museum of Fine Arts", lat: 42.3381, lng: -71.0948 },
    { name: "Northeastern University", lat: 42.3403, lng: -71.0889 },
    { name: "Symphony", lat: 42.3429, lng: -71.0853 },
    { name: "Prudential", lat: 42.3457, lng: -71.0816 },
    { name: "Copley", lat: 42.3499, lng: -71.0778 },
    { name: "Arlington", lat: 42.3524, lng: -71.0704 },
    { name: "Boylston", lat: 42.3530, lng: -71.0646 },
    { name: "Park Street", lat: 42.3563, lng: -71.0622 },
  ],
};

function App() {
  const [selectedLine, setSelectedLine] = useState('Green-B')
  const [trustworthiness, setTrustworthiness] = useState(85)
  const [selectedStation, setSelectedStation] = useState("Park Street")
  const [logs, setLogs] = useState<LogEntry[]>([
    {
      id: 1,
      timestamp: new Date().toLocaleTimeString(),
      message: "Application initialized",
    }
  ])

  const currentStations = stationsByLine[selectedLine]

  const addLog = (message: string) => {
    const newLog: LogEntry = {
      id: Date.now(),
      timestamp: new Date().toLocaleTimeString(),
      message,
    }
    setLogs(prev => [newLog, ...prev].slice(0, 50)) // Keep only last 50 logs
  }

  const handleStationSelect = (station: string) => {
    setSelectedStation(station)
    setTrustworthiness(Math.floor(Math.random() * 100)) // Random percentage for demo
    addLog(`Station selected: ${station}`)
  }

  const handleLineChange = (line: string) => {
    setSelectedLine(line)
    addLog(`Switched to ${line}`)
  }

  return (
    <div className="app-container">
      <div className="left-panel">
        <div className="trustworthiness-display">
          <h1 className="station-name">{selectedStation}</h1>
          <div className="percentage-circle">
            <span className="percentage-number">{trustworthiness}%</span>
          </div>
          <p className="trustworthiness-label">Trustworthiness</p>
        </div>
      </div>
      
      <div className="right-panel">
        <div className="map-container">
          <div className="map-header">
            <h2>Green Line</h2>
            <select 
              className="line-selector" 
              value={selectedLine} 
              onChange={(e) => handleLineChange(e.target.value)}
            >
              <option value="Green-B">Green Line B</option>
              <option value="Green-C">Green Line C</option>
              <option value="Green-D">Green Line D</option>
              <option value="Green-E">Green Line E</option>
            </select>
          </div>
          <div className="leaflet-map-container">
            <MapContainer
              center={[42.348, -71.115]}
              zoom={13}
              style={{ height: '100%', width: '100%', borderRadius: '20px' }}
            >
              <TileLayer
                attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
                url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
              />
              {currentStations.map((station) => (
                <CircleMarker
                  key={station.name}
                  center={[station.lat, station.lng]}
                  radius={8}
                  fillColor="#00843D"
                  color="white"
                  weight={3}
                  fillOpacity={1}
                  eventHandlers={{
                    click: () => {
                      handleStationSelect(station.name);
                    },
                  }}
                >
                  <Popup>
                    <strong>{station.name}</strong>
                  </Popup>
                </CircleMarker>
              ))}
            </MapContainer>
          </div>
        </div>
        
        <div className="log-container">
          <h3>Activity Log</h3>
          <div className="log-content">
            {logs.map(log => (
              <div key={log.id} className="log-entry">
                <span className="log-timestamp">[{log.timestamp}]</span>
                <span className="log-message">{log.message}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}

export default App

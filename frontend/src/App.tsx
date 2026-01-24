import { useState } from 'react'
import './App.css'

interface LogEntry {
  id: number;
  timestamp: string;
  message: string;
  elementId?: string;
  elementClass?: string;
}

// Green Line B stations with actual GPS coordinates
const greenLineBStations = [
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
];

// Map bounds for Boston area (approximate based on the embedded map)
const mapBounds = {
  north: 42.400,
  south: 42.320,
  west: -71.180,
  east: -71.040
};

// Convert lat/lng to percentage position on the map
const getMapPosition = (lat: number, lng: number) => {
  const leftPercent = ((lng - mapBounds.west) / (mapBounds.east - mapBounds.west)) * 100;
  const topPercent = ((mapBounds.north - lat) / (mapBounds.north - mapBounds.south)) * 100;
  return {
    left: `${leftPercent.toFixed(1)}%`,
    top: `${topPercent.toFixed(1)}%`
  };
};

function App() {
  const [trustworthiness, setTrustworthiness] = useState(85)
  const [selectedStation, setSelectedStation] = useState("Park Street")
  const [logs, setLogs] = useState<LogEntry[]>([
    {
      id: 1,
      timestamp: new Date().toLocaleTimeString(),
      message: "Application initialized",
    }
  ])

  const addLog = (message: string, elementId?: string, elementClass?: string) => {
    const newLog: LogEntry = {
      id: Date.now(),
      timestamp: new Date().toLocaleTimeString(),
      message,
      elementId,
      elementClass
    }
    setLogs(prev => [newLog, ...prev].slice(0, 50)) // Keep only last 50 logs
  }

  const handleMapClick = (event: React.MouseEvent<HTMLDivElement>) => {
    const target = event.target as HTMLElement
    addLog(
      "Map clicked", 
      target.id || "no-id", 
      target.className || "no-class"
    )
  }

  const handleStationSelect = (station: string) => {
    setSelectedStation(station)
    setTrustworthiness(Math.floor(Math.random() * 100)) // Random percentage for demo
    addLog(`Station selected: ${station}`)
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
          <h2>Green Line B</h2>
          <div 
            className="google-map-container" 
            id="station-map"
            onClick={handleMapClick}
          >
            <iframe
              id="google-maps-iframe"
              className="google-maps-embed"
              src="https://www.google.com/maps/embed?pb=!1m18!1m12!1m3!1d94459.28756345654!2d-71.1569722!3d42.3600825!2m3!1f0!2f0!3f0!3m2!1i1024!2i768!4f13.1!3m3!1m2!1s0x89e3652d0d3d311b%3A0x787cbf240162e8a0!2sBoston%2C%20MA!5e0!3m2!1sen!2sus!4v1635000000000!5m2!1sen!2sus"
              allowFullScreen
              loading="lazy"
              referrerPolicy="no-referrer-when-downgrade"
              title="MBTA Stations Map"
            ></iframe>
            <div className="map-overlay" onClick={handleMapClick}>
              <div className="station-markers">
                {greenLineBStations.map((station, index) => {
                  const position = getMapPosition(station.lat, station.lng);
                  return (
                    <div 
                      key={station.name}
                      className="station-marker green-b-station" 
                      id={`station-${index}`}
                      style={{
                        left: position.left,
                        top: position.top
                      }}
                      onClick={(e) => {
                        e.stopPropagation();
                        handleStationSelect(station.name);
                      }}
                      title={station.name}
                    >
                      <span className="marker-label">{station.name}</span>
                    </div>
                  );
                })}
              </div>
            </div>
          </div>
        </div>
        
        <div className="log-container">
          <h3>Activity Log</h3>
          <div className="log-content">
            {logs.map(log => (
              <div key={log.id} className="log-entry">
                <span className="log-timestamp">[{log.timestamp}]</span>
                <span className="log-message">{log.message}</span>
                {log.elementId && (
                  <span className="log-details">
                    {" "}(ID: {log.elementId}, Class: {log.elementClass})
                  </span>
                )}
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}

export default App

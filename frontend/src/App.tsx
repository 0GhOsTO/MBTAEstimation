import { useState } from 'react'
import './App.css'

interface LogEntry {
  id: number;
  timestamp: string;
  message: string;
  elementId?: string;
  elementClass?: string;
}

function App() {
  const [trustworthiness, setTrustworthiness] = useState(85)
  const [selectedStation, setSelectedStation] = useState("North Station")
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
          <h2>MBTA Station Map</h2>
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
                <div 
                  className="station-marker north-station" 
                  id="north-station-marker"
                  onClick={(e) => {
                    e.stopPropagation();
                    handleStationSelect("North Station");
                  }}
                  title="North Station"
                >
                  <span className="marker-label">North Station</span>
                </div>
                <div 
                  className="station-marker south-station" 
                  id="south-station-marker"
                  onClick={(e) => {
                    e.stopPropagation();
                    handleStationSelect("South Station");
                  }}
                  title="South Station"
                >
                  <span className="marker-label">South Station</span>
                </div>
                <div 
                  className="station-marker back-bay" 
                  id="back-bay-marker"
                  onClick={(e) => {
                    e.stopPropagation();
                    handleStationSelect("Back Bay");
                  }}
                  title="Back Bay"
                >
                  <span className="marker-label">Back Bay</span>
                </div>
                <div 
                  className="station-marker downtown-crossing" 
                  id="downtown-crossing-marker"
                  onClick={(e) => {
                    e.stopPropagation();
                    handleStationSelect("Downtown Crossing");
                  }}
                  title="Downtown Crossing"
                >
                  <span className="marker-label">Downtown Crossing</span>
                </div>
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

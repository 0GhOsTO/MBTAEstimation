import { useState, useEffect } from 'react'
import { MapContainer, TileLayer, CircleMarker, Popup } from 'react-leaflet'
import 'leaflet/dist/leaflet.css'
import 'katex/dist/katex.min.css'
import katex from 'katex'
import './App.css'

// LatexMath component for rendering LaTeX
function LatexMath({ children, block = true }: { children: string; block?: boolean }) {
  const html = katex.renderToString(children, {
    throwOnError: false,
    displayMode: block,
  })
  
  return <div dangerouslySetInnerHTML={{ __html: html }} />
}

interface LogEntry {
  id: number;
  timestamp: string;
  message: string;
}

interface StationStats {
  station_id: string;
  station_name: string;
  inbound_accuracy: number;
  inbound_total: number;
  outbound_accuracy: number;
  outbound_total: number;
}

// Green Line stations with actual GPS coordinates
const stationsByLine: { [key: string]: Array<{ name: string; lat: number; lng: number }> } = {
  'Green-B': [
    { name: "Boston College", lat: 42.3396, lng: -71.1686 },
    { name: "South Street", lat: 42.3399, lng: -71.1571 },
    { name: "Chestnut Hill Ave", lat: 42.3387, lng: -71.1527 },
    { name: "Chiswick Road", lat: 42.3406, lng: -71.1504 },
    { name: "Sutherland Road", lat: 42.3410, lng: -71.1464 },
    { name: "Washington Street", lat: 42.3431, lng: -71.1420 },
    { name: "Warren Street", lat: 42.3485, lng: -71.1401 },
    { name: "Allston Street", lat: 42.3484, lng: -71.1373 },
    { name: "Griggs Street", lat: 42.3481, lng: -71.1345 },
    { name: "Harvard Avenue", lat: 42.3502, lng: -71.1312 },
    { name: "Packards Corner", lat: 42.3519, lng: -71.1251 },
    { name: "Babcock Street", lat: 42.3513, lng: -71.1218 },
    { name: "Pleasant Street", lat: 42.3513, lng: -71.1187 },
    { name: "Saint Paul Street", lat: 42.3511, lng: -71.1157 },
    { name: "BU West", lat: 42.3499, lng: -71.1138 },
    { name: "BU Central", lat: 42.3497, lng: -71.1070 },
    { name: "BU East", lat: 42.3496, lng: -71.1040 },
    { name: "Blandford Street", lat: 42.3493, lng: -71.1002 },
    { name: "Kenmore", lat: 42.3488, lng: -71.0952 },
    { name: "Hynes Convention Center", lat: 42.3472, lng: -71.0876 },
    { name: "Copley", lat: 42.3499, lng: -71.0778 },
    { name: "Arlington", lat: 42.3524, lng: -71.0704 },
    { name: "Boylston", lat: 42.3530, lng: -71.0646 },
    { name: "Park Street", lat: 42.3563, lng: -71.0622 },
    { name: "Government Center", lat: 42.3597, lng: -71.0592 },
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
    { name: "Government Center", lat: 42.3597, lng: -71.0592 },
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
    { name: "Government Center", lat: 42.3597, lng: -71.0592 },
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
    { name: "Government Center", lat: 42.3597, lng: -71.0592 },
  ],
};

// Map station names to backend station IDs (must match backend exactly)
const stationNameToID: { [key: string]: string } = {
  "Boston College": "place-lake",
  "South Street": "place-sougr",
  "Chestnut Hill Ave": "place-chill",
  "Chiswick Road": "place-chswk",
  "Sutherland Road": "place-sthld",
  "Washington Street": "place-wascm",
  "Warren Street": "place-wrnst",
  "Allston Street": "place-alsgr",
  "Griggs Street": "place-grigg",
  "Harvard Avenue": "place-harvd",
  "Packards Corner": "place-brico",
  "Babcock Street": "70136",  // Orphan platform
  "Pleasant Street": "70138",  // Orphan platform
  "Saint Paul Street": "70140",  // Orphan platform
  "BU West": "70142",  // Orphan platform
  "BU Central": "place-bucen",
  "BU East": "place-buest",
  "Blandford Street": "place-bland",
  "Kenmore": "place-kencl",
  "Hynes Convention Center": "place-hymnl",
  "Copley": "place-coecl",
  "Arlington": "place-armnl",
  "Boylston": "place-boyls",
  "Park Street": "place-pktrm",
  "Government Center": "place-gover",
};

function App() {
  const [selectedLine, setSelectedLine] = useState('Green-B')
  const [inboundAccuracy, setInboundAccuracy] = useState(0)
  const [outboundAccuracy, setOutboundAccuracy] = useState(0)
  const [selectedStation, setSelectedStation] = useState("Park Street")
  const [stationData, setStationData] = useState<{ [key: string]: StationStats }>({})
  const [showEquation, setShowEquation] = useState(false)
  const [logs, setLogs] = useState<LogEntry[]>([
    {
      id: 1,
      timestamp: new Date().toLocaleTimeString(),
      message: "Application initialized",
    }
  ])

  // Fetch statistics from backend
  useEffect(() => {
    const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'
    
    const fetchStatistics = async () => {
      try {
        const response = await fetch(`${API_URL}/api/statistics`)
        const data: StationStats[] = await response.json()
        
        // Convert array to map for easy lookup
        const dataMap: { [key: string]: StationStats } = {}
        data.forEach(stat => {
          dataMap[stat.station_id] = stat
        })
        setStationData(dataMap)
        
        // Update current station if it has data
        const stationID = stationNameToID[selectedStation]
        if (stationID && dataMap[stationID]) {
          updateAccuracyForStation(dataMap[stationID])
        } else {
          updateAccuracyForStation(undefined)
        }
        
        addLog(`Fetched statistics for ${data.length} stations`)
      } catch (error) {
        console.error('Error fetching statistics:', error)
        addLog('Error: Could not connect to backend')
      }
    }

    // Fetch immediately
    fetchStatistics()

    // Fetch every 10 seconds
    const interval = setInterval(fetchStatistics, 10000)

    return () => clearInterval(interval)
  }, [selectedStation])

  const currentStations = stationsByLine[selectedLine]

  const addLog = (message: string) => {
    const newLog: LogEntry = {
      id: Date.now(),
      timestamp: new Date().toLocaleTimeString(),
      message,
    }
    setLogs(prev => [newLog, ...prev].slice(0, 50)) // Keep only last 50 logs
  }

  const updateAccuracyForStation = (stats?: StationStats) => {
    if (!stats) {
      setInboundAccuracy(0)
      setOutboundAccuracy(0)
      return
    }

    setInboundAccuracy(Math.round(stats.inbound_accuracy))
    setOutboundAccuracy(Math.round(stats.outbound_accuracy))
  }

  const handleStationSelect = (station: string) => {
    setSelectedStation(station)
    
    // Get station ID from name
    const stationID = stationNameToID[station]
    
    if (stationID && stationData[stationID]) {
      const stats = stationData[stationID]
      updateAccuracyForStation(stats)
      addLog(`Station: ${station} | Inbound: ${stats.inbound_total} predictions | Outbound: ${stats.outbound_total} predictions`)
    } else {
      // No data yet
      updateAccuracyForStation(undefined)
      addLog(`${station} - No data available yet`)
    }
  }

  const handleLineChange = (line: string) => {
    setSelectedLine(line)
    addLog(`Switched to ${line}`)
  }

  const selectedStationID = stationNameToID[selectedStation]
  const selectedStats = selectedStationID ? stationData[selectedStationID] : undefined
  const inboundTotal = selectedStats ? selectedStats.inbound_total : 0
  const outboundTotal = selectedStats ? selectedStats.outbound_total : 0

  return (
    <div className="app-container">
      <div className="left-panel">
        <div className="trustworthiness-display">
          <h1 className="project-title">MBTA Reliability</h1>
          <h2 className="station-name">{selectedStation}</h2>
          <button 
            className="equation-toggle-btn"
            onClick={() => setShowEquation(!showEquation)}
          >
            <span>{showEquation ? '▼' : '▶'}</span>
            <span>How it's calculated</span>
          </button>
          {showEquation && (
            <div className="equation-box">
              <div className="equation-line">
                <LatexMath>
                  {`\\text{Trustworthiness}(\\%) = \\frac{\\text{Correct Predictions}}{\\text{Total Predictions}} \\times 100`}
                </LatexMath>
              </div>
              <div className="equation-line">
                <LatexMath>
                  {`\\text{Correct} = \\begin{cases} 1 & \\text{if } |\\text{Predicted} - \\text{Actual}| \\leq 3\\text{ min} \\\\ 0 & \\text{otherwise} \\end{cases}`}
                </LatexMath>
              </div>
            </div>
          )}
          <div className="metrics-and-stations">
            <div className="accuracy-circles-container">
              <div className="accuracy-item">
                <div className={`percentage-circle ${inboundTotal > 0 ? (inboundAccuracy >= 50 ? 'good' : 'poor') : 'no-data'}`}>
                  <span className="percentage-number">
                    {inboundTotal > 0 ? `${inboundAccuracy}%` : 'N/A'}
                  </span>
                </div>
                <p className="accuracy-label">Inbound</p>
                {inboundTotal > 0 ? <p className="prediction-count">({inboundTotal} predictions)</p> : null}
              </div>
              <div className="accuracy-item">
                <div className={`percentage-circle ${outboundTotal > 0 ? (outboundAccuracy >= 50 ? 'good' : 'poor') : 'no-data'}`}>
                  <span className="percentage-number">
                    {outboundTotal > 0 ? `${outboundAccuracy}%` : 'N/A'}
                  </span>
                </div>
                <p className="accuracy-label">Outbound</p>
                {outboundTotal > 0 ? <p className="prediction-count">({outboundTotal} predictions)</p> : null}
              </div>
            </div>
          </div>
          <div className="station-selector-container">
            <select 
              className="station-selector" 
              value={selectedStation} 
              onChange={(e) => handleStationSelect(e.target.value)}
            >
              {currentStations.map((station) => (
                <option key={station.name} value={station.name}>
                  {station.name}
                </option>
              ))}
            </select>
          </div>
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

import { useState, useEffect } from 'react'
import { MapContainer, TileLayer, CircleMarker, Popup, useMap } from 'react-leaflet'
import 'leaflet/dist/leaflet.css'
import './App.css'

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
  inbound_recent_diff_minutes?: number | null;
  outbound_recent_diff_minutes?: number | null;
}

// Green Line stations with actual GPS coordinates
const stationsByLine: { [key: string]: Array<{ name: string; lat: number; lng: number }> } = {
  'Green-B': [
    { name: "Boston College", lat: 42.340081, lng: -71.166769 },
    { name: "South Street", lat: 42.339600, lng: -71.157661 },
    { name: "Chestnut Hill Ave", lat: 42.338169, lng: -71.153160 },
    { name: "Chiswick Road", lat: 42.340805, lng: -71.150711 },
    { name: "Sutherland Road", lat: 42.341614, lng: -71.146202 },
    { name: "Washington Street", lat: 42.343864, lng: -71.142853 },
    { name: "Warren Street", lat: 42.348343, lng: -71.140457 },
    { name: "Allston Street", lat: 42.348701, lng: -71.137955 },
    { name: "Griggs Street", lat: 42.348545, lng: -71.134949 },
    { name: "Harvard Avenue", lat: 42.350243, lng: -71.131355 },
    { name: "Packards Corner", lat: 42.351967, lng: -71.125031 },
    { name: "Babcock Street", lat: 42.351538, lng: -71.119553 },
    { name: "Amory Street", lat: 42.350901, lng: -71.114318 },
    { name: "BU Central", lat: 42.350082, lng: -71.106865 },
    { name: "BU East", lat: 42.349735, lng: -71.103889 },
    { name: "Blandford Street", lat: 42.349293, lng: -71.100258 },
    { name: "Kenmore", lat: 42.348949, lng: -71.095169 },
    { name: "Hynes Convention Center", lat: 42.347888, lng: -71.087903 },
    { name: "Copley", lat: 42.349974, lng: -71.077447 },
    { name: "Arlington", lat: 42.351902, lng: -71.070893 },
    { name: "Boylston", lat: 42.353020, lng: -71.064590 },
    { name: "Park Street", lat: 42.356395, lng: -71.062424 },
    { name: "Government Center", lat: 42.359705, lng: -71.059215 },
  ],
  'Green-C': [
    { name: "Cleveland Circle", lat: 42.336142, lng: -71.149326 },
    { name: "Englewood Ave", lat: 42.336971, lng: -71.145660 },
    { name: "Dean Road", lat: 42.337807, lng: -71.141853 },
    { name: "Tappan Street", lat: 42.338459, lng: -71.138702 },
    { name: "Washington Square", lat: 42.339394, lng: -71.135330 },
    { name: "Fairbanks Street", lat: 42.339725, lng: -71.131073 },
    { name: "Brandon Hall", lat: 42.340023, lng: -71.129082 },
    { name: "Summit Avenue", lat: 42.341110, lng: -71.125610 },
    { name: "Coolidge Corner", lat: 42.342116, lng: -71.121263 },
    { name: "Kent Street", lat: 42.344074, lng: -71.114197 },
    { name: "Hawes Street", lat: 42.344906, lng: -71.111145 },
    { name: "Saint Mary's Street", lat: 42.345974, lng: -71.107353 },
    { name: "Saint Paul Street", lat: 42.343327, lng: -71.116997 },
    { name: "Kenmore", lat: 42.348949, lng: -71.095169 },
    { name: "Hynes Convention Center", lat: 42.347888, lng: -71.087903 },
    { name: "Copley", lat: 42.349974, lng: -71.077447 },
    { name: "Arlington", lat: 42.351902, lng: -71.070893 },
    { name: "Boylston", lat: 42.353020, lng: -71.064590 },
    { name: "Park Street", lat: 42.356395, lng: -71.062424 },
    { name: "Government Center", lat: 42.359705, lng: -71.059215 },
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
    { name: "Haymarket", lat: 42.363021, lng: -71.058290 },
    { name: "North Station", lat: 42.365577, lng: -71.061290 },
    { name: "Science Park/West End", lat: 42.366664, lng: -71.067666 },
    { name: "Lechmere", lat: 42.371572, lng: -71.076584 },
    { name: "Union Square", lat: 42.377359, lng: -71.094761 },
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
    { name: "Haymarket", lat: 42.363021, lng: -71.058290 },
    { name: "North Station", lat: 42.365577, lng: -71.061290 },
    { name: "Science Park/West End", lat: 42.366664, lng: -71.067666 },
    { name: "Lechmere", lat: 42.371572, lng: -71.076584 },
    { name: "East Somerville", lat: 42.379467, lng: -71.086625 },
    { name: "Gilman Square", lat: 42.387928, lng: -71.096766 },
    { name: "Magoun Square", lat: 42.393682, lng: -71.106388 },
    { name: "Ball Square", lat: 42.399889, lng: -71.111003 },
    { name: "Medford/Tufts", lat: 42.407975, lng: -71.117044 },
  ],
};

const lineDirectionLabels: { [key: string]: { inbound: string; outbound: string } } = {
  'Green-B': { inbound: 'to Government Center', outbound: 'to Boston College' },
  'Green-C': { inbound: 'to Government Center', outbound: 'to Cleveland Circle' },
  'Green-D': { inbound: 'to Union Square', outbound: 'to Riverside' },
  'Green-E': { inbound: 'to Medford/Tufts', outbound: 'to Heath Street' },
}

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
  "Babcock Street": "place-babck",
  "Amory Street": "place-amory",
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
  "Cleveland Circle": "place-clmnl",
  "Englewood Ave": "place-engav",
  "Dean Road": "place-denrd",
  "Tappan Street": "place-tapst",
  "Washington Square": "place-bcnwa",
  "Fairbanks Street": "place-fbkst",
  "Brandon Hall": "place-bndhl",
  "Summit Avenue": "place-sumav",
  "Coolidge Corner": "place-cool",
  "Kent Street": "place-kntst",
  "Hawes Street": "place-hwsst",
  "Saint Mary's Street": "place-smary",
  "Saint Paul Street": "place-stpul",
  // Green-D unique stations
  "Riverside": "place-river",
  "Woodland": "place-woodl",
  "Waban": "place-waban",
  "Eliot": "place-eliot",
  "Newton Highlands": "place-newtn",
  "Newton Centre": "place-newto",
  "Chestnut Hill": "place-chhil",
  "Reservoir": "place-rsmnl",
  "Beaconsfield": "place-bcnfd",
  "Brookline Hills": "place-brkhl",
  "Brookline Village": "place-bvmnl",
  "Longwood": "place-longw",
  "Fenway": "place-fenwy",
  // Green-E unique stations
  "Heath Street": "place-hsmnl",
  "Back of the Hill": "place-bckhl",
  "Riverway": "place-rvrwy",
  "Mission Park": "place-mispk",
  "Fenwood Road": "place-fenwd",
  "Brigham Circle": "place-brmnl",
  "Longwood Medical Area": "place-lngmd",
  "Museum of Fine Arts": "place-mfa",
  "Northeastern University": "place-nuniv",
  "Symphony": "place-symcl",
  "Prudential": "place-prmnl",
  "Haymarket": "place-haecl",
  "North Station": "place-north",
  "Science Park/West End": "place-spmnl",
  "Lechmere": "place-lech",
  "Union Square": "place-unsqu",
  "East Somerville": "place-esomr",
  "Gilman Square": "place-gilmn",
  "Magoun Square": "place-mgngl",
  "Ball Square": "place-balsq",
  "Medford/Tufts": "place-mdftf",
};

const supportedLines = ['Green-B', 'Green-C', 'Green-D', 'Green-E'] as const
const STORAGE_KEY = 'mbta-reliability-ui-state'

interface PersistedUIState {
  selectedLine: string
  selectedStation: string
}

const getInitialUIState = (): PersistedUIState => {
  const fallbackState: PersistedUIState = {
    selectedLine: 'Green-B',
    selectedStation: 'Park Street',
  }

  if (typeof window === 'undefined') {
    return fallbackState
  }

  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) {
      return fallbackState
    }

    const parsed = JSON.parse(raw) as Partial<PersistedUIState>
    const line = parsed.selectedLine
    const station = parsed.selectedStation

    if (!line || !stationsByLine[line]) {
      return fallbackState
    }

    const stationExistsOnLine = stationsByLine[line].some((s) => s.name === station)
    if (!station || !stationExistsOnLine) {
      return {
        selectedLine: line,
        selectedStation: stationsByLine[line][0].name,
      }
    }

    return {
      selectedLine: line,
      selectedStation: station,
    }
  } catch {
    return fallbackState
  }
}

function App() {
  const initialUIState = getInitialUIState()
  const [selectedLine, setSelectedLine] = useState(initialUIState.selectedLine)
  const [inboundAccuracy, setInboundAccuracy] = useState(0)
  const [outboundAccuracy, setOutboundAccuracy] = useState(0)
  const [selectedStation, setSelectedStation] = useState(initialUIState.selectedStation)
  const [stationData, setStationData] = useState<{ [key: string]: StationStats }>({})
  const [allLinesData, setAllLinesData] = useState<{ [line: string]: { [stationID: string]: StationStats } }>({})
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
        const response = await fetch(`${API_URL}/api/statistics?route=${encodeURIComponent(selectedLine)}`)
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
  }, [selectedLine, selectedStation])

  // Fetch stats for all lines in background so shared-station badges can show their accuracy colors
  useEffect(() => {
    const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'
    const fetchAllLines = async () => {
      const results: { [line: string]: { [stationID: string]: StationStats } } = {}
      await Promise.all(
        supportedLines.map(async (line) => {
          try {
            const resp = await fetch(`${API_URL}/api/statistics?route=${encodeURIComponent(line)}`)
            const data: StationStats[] = await resp.json()
            const map: { [stationID: string]: StationStats } = {}
            data.forEach(s => { map[s.station_id] = s })
            results[line] = map
          } catch {
            // ignore per-line failures
          }
        })
      )
      setAllLinesData(results)
    }
    fetchAllLines()
    const interval = setInterval(fetchAllLines, 10000)
    return () => clearInterval(interval)
  }, [])

  const getBadgeClass = (line: string, direction: 'inbound' | 'outbound') => {
    const stationID = stationNameToID[selectedStation]
    const stats = allLinesData[line]?.[stationID]
    if (!stats) return 'no-data'
    const accuracy = direction === 'inbound'
      ? (stats.inbound_total > 0 ? stats.inbound_accuracy : null)
      : (stats.outbound_total > 0 ? stats.outbound_accuracy : null)
    if (accuracy === null) return 'no-data'
    if (accuracy >= 70) return 'good'
    if (accuracy >= 50) return 'moderate'
    return 'poor'
  }

  const currentStations = stationsByLine[selectedLine]
  const lineMapView: { [key: string]: { center: [number, number]; zoom: number } } = {
    'Green-B': { center: [42.349, -71.124], zoom: 13 },
    'Green-C': { center: [42.344, -71.115], zoom: 13 },
    'Green-D': { center: [42.344, -71.148], zoom: 12 },
    'Green-E': { center: [42.363, -71.093], zoom: 12 },
  }
  const mapView = lineMapView[selectedLine] ?? { center: [42.348, -71.115], zoom: 13 }

  useEffect(() => {
    try {
      window.localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify({ selectedLine, selectedStation }),
      )
    } catch {
      // Ignore storage write errors (private browsing, quota, etc).
    }
  }, [selectedLine, selectedStation])

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

  const formatRecentDiff = (diff?: number | null) => {
    if (diff === null || diff === undefined || Number.isNaN(diff)) {
      return 'Waiting...'
    }
    const absDiff = Math.abs(diff)
    if (absDiff < 0.05) {
      return 'Train arrived on time'
    }
    const roundedUp = Math.ceil(absDiff)
    const unit = roundedUp === 1 ? 'minute' : 'minutes'
    return diff > 0
      ? `Train arrived ${roundedUp} ${unit} late`
      : `Train arrived ${roundedUp} ${unit} fast`
  }

  const getRecentDiffClass = (diff?: number | null) => {
    if (diff === null || diff === undefined || Number.isNaN(diff)) {
      return 'no-data'
    }
    if (Math.abs(diff) < 0.05) {
      return 'on-time'
    }
    return diff > 0 ? 'late' : 'early'
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
    const firstStation = stationsByLine[line]?.[0]?.name
    if (firstStation) {
      setSelectedStation(firstStation)
    }
    addLog(`Switched to ${line}`)
  }

  const selectedStationID = stationNameToID[selectedStation]
  const selectedStats = selectedStationID ? stationData[selectedStationID] : undefined
  const inboundTotal = selectedStats ? selectedStats.inbound_total : 0
  const outboundTotal = selectedStats ? selectedStats.outbound_total : 0
  const inboundRecentDiff = selectedStats?.inbound_recent_diff_minutes
  const outboundRecentDiff = selectedStats?.outbound_recent_diff_minutes

  // Lines that also serve the currently selected station (excluding the active line)
  const otherLinesAtStation = Object.entries(stationsByLine)
    .filter(([line, stations]) => line !== selectedLine && stations.some(s => s.name === selectedStation))
    .map(([line]) => line)

  return (
    <div className="app-container">
      <div className="left-panel">
        <div className="trustworthiness-display">
          <h1 className="project-title">MBTA Reliability</h1>
          <div className="selector-container">
            <select
              className="line-selector"
              value={selectedLine}
              onChange={(e) => handleLineChange(e.target.value)}
            >
              {supportedLines.map((line) => (
                <option key={line} value={line}>{line.replace('-', ' ')}</option>
              ))}
            </select>
            <span className="selector-separator">:</span>
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
          <h2 className="station-name">{selectedStation}</h2>
          <div className="metrics-and-stations">
            <div className="accuracy-circles-container">
              <div className="accuracy-item">
                <div className={`percentage-circle ${inboundTotal > 0 ? (inboundAccuracy >= 70 ? 'good' : inboundAccuracy >= 50 ? 'moderate' : 'poor') : 'no-data'}`}>
                  <span className="percentage-number">
                    {inboundTotal > 0 ? `${inboundAccuracy}%` : 'N/A'}
                  </span>
                </div>
                {otherLinesAtStation.length > 0 && (
                  <div className="shared-line-badges">
                    {otherLinesAtStation.map(line => (
                      <span key={line} className={`line-badge ${getBadgeClass(line, 'inbound')}`}>
                        {line.replace('Green-', '')}
                      </span>
                    ))}
                  </div>
                )}
                <p className="accuracy-label">Inbound</p>
                <p className="direction-label">{lineDirectionLabels[selectedLine]?.inbound ?? 'to Government Center'}</p>
                <p className="prediction-count">({inboundTotal} predictions)</p>
              </div>
              <div className="accuracy-item">
                <div className={`percentage-circle ${outboundTotal > 0 ? (outboundAccuracy >= 70 ? 'good' : outboundAccuracy >= 50 ? 'moderate' : 'poor') : 'no-data'}`}>
                  <span className="percentage-number">
                    {outboundTotal > 0 ? `${outboundAccuracy}%` : 'N/A'}
                  </span>
                </div>
                {otherLinesAtStation.length > 0 && (
                  <div className="shared-line-badges">
                    {otherLinesAtStation.map(line => (
                      <span key={line} className={`line-badge ${getBadgeClass(line, 'outbound')}`}>
                        {line.replace('Green-', '')}
                      </span>
                    ))}
                  </div>
                )}
                <p className="accuracy-label">Outbound</p>
                <p className="direction-label">{lineDirectionLabels[selectedLine]?.outbound ?? 'to Outbound Terminal'}</p>
                <p className="prediction-count">({outboundTotal} predictions)</p>
              </div>
            </div>
          </div>
          <div className="recent-diff-section">
            <h3>Most Recent Arrival</h3>
            <div className="recent-diff-grid">
              <div className={`recent-diff-card ${getRecentDiffClass(inboundRecentDiff)}`}>
                <p className="recent-diff-direction">Inbound</p>
                <p className="recent-diff-value">{formatRecentDiff(inboundRecentDiff)}</p>
              </div>
              <div className={`recent-diff-card ${getRecentDiffClass(outboundRecentDiff)}`}>
                <p className="recent-diff-direction">Outbound</p>
                <p className="recent-diff-value">{formatRecentDiff(outboundRecentDiff)}</p>
              </div>
            </div>
          </div>
          <button 
            className="equation-toggle-btn"
            onClick={() => setShowEquation(!showEquation)}
          >
            <span>{showEquation ? '▼' : '▶'}</span>
            <span>What does the % mean?</span>
          </button>
          {showEquation && (
            <div className="equation-box">
              <div className="reliability-explanation">
                <div className="reliability-levels">
                  <div className="reliability-level level-poor">
                    <span className="level-range">~50%</span>
                    <span className="level-label">Train often delayed</span>
                  </div>
                  <div className="reliability-level level-moderate">
                    <span className="level-range">50-70%</span>
                    <span className="level-label">Train sometimes delayed</span>
                  </div>
                  <div className="reliability-level level-good">
                    <span className="level-range">70%+</span>
                    <span className="level-label">Train usually on time</span>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
      
      <div className="right-panel">
        <div className="map-container">
          <div className="map-header">
            <h2>Station Map</h2>
          </div>
          <div className="leaflet-map-container">
            <MapContainer
              center={[42.348, -71.115]}
              zoom={13}
              style={{ height: '100%', width: '100%', borderRadius: '20px' }}
            >
              <MapViewportController center={mapView.center} zoom={mapView.zoom} />
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

function MapViewportController({ center, zoom }: { center: [number, number]; zoom: number }) {
  const map = useMap()

  useEffect(() => {
    map.flyTo(center, zoom, { duration: 0.6 })
  }, [map, center, zoom])

  return null
}

export default App

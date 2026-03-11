import { useState, useEffect, useMemo, useRef } from 'react'
import { MapContainer, TileLayer, CircleMarker, Popup } from 'react-leaflet'
import 'leaflet/dist/leaflet.css'
import './App.css'

<<<<<<< ours
<<<<<<< ours
<<<<<<< ours
=======
=======
>>>>>>> theirs
=======
>>>>>>> theirs
function LatexMath({ children, block = true }: { children: string; block?: boolean }) {
  const html = katex.renderToString(children, {
    throwOnError: false,
    displayMode: block,
  })

  return <div dangerouslySetInnerHTML={{ __html: html }} />
}

>>>>>>> theirs
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
    { name: "Amory Street", lat: 42.3511, lng: -71.1157 },
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
}

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
<<<<<<< ours
<<<<<<< ours
<<<<<<< ours
  "Babcock Street": "70136",  // Orphan platform
  "Amory Street": "70140",  // Orphan platform
=======
=======
>>>>>>> theirs
=======
>>>>>>> theirs
  "Babcock Street": "70136",
  "Pleasant Street": "70138",
  "Saint Paul Street": "70140",
  "BU West": "70142",
<<<<<<< ours
<<<<<<< ours
>>>>>>> theirs
=======
>>>>>>> theirs
=======
>>>>>>> theirs
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
}

const supportedLines = ['Green-B'] as const
const STORAGE_KEY = 'mbta-reliability-ui-state'

interface PersistedUIState {
  selectedLine: string
  selectedStation: string
}

const getApiBaseUrl = (): string => {
  const configured = import.meta.env.VITE_API_URL
  if (configured && configured.trim()) {
    return configured.trim().replace(/\/$/, '')
  }

  if (import.meta.env.DEV) {
    return 'http://localhost:8080'
  }

  if (typeof window !== 'undefined') {
    return window.location.origin
  }

  return ''
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
  const [showEquation, setShowEquation] = useState(false)
  const [logs, setLogs] = useState<LogEntry[]>([
    {
      id: 1,
      timestamp: new Date().toLocaleTimeString(),
      message: "Application initialized",
    }
  ])
  const lastFetchMessageRef = useRef('')

  const API_URL = useMemo(() => getApiBaseUrl(), [])
  const STATS_PATH = useMemo(() => (import.meta.env.VITE_STATS_PATH || '/api/v1/statistics').trim(), [])
  const POLL_MS = Number(import.meta.env.VITE_STATS_POLL_MS || 10000)
  const REQUEST_TIMEOUT_MS = Number(import.meta.env.VITE_STATS_TIMEOUT_MS || 8000)

  const addLog = (message: string) => {
    const newLog: LogEntry = {
      id: Date.now(),
      timestamp: new Date().toLocaleTimeString(),
      message,
    }
    setLogs(prev => [newLog, ...prev].slice(0, 50))
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

  useEffect(() => {
    const fetchStatistics = async () => {
      const controller = new AbortController()
      const timeout = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS)

      try {
        const response = await fetch(`${API_URL}${STATS_PATH}`, { signal: controller.signal })
        if (!response.ok) {
          throw new Error(`statistics request failed: ${response.status}`)
        }

        const payload = await response.json()
        if (!Array.isArray(payload)) {
          throw new Error('statistics response format is invalid')
        }

        const dataMap: { [key: string]: StationStats } = {}
        for (const stat of payload as StationStats[]) {
          dataMap[stat.station_id] = stat
        }
        setStationData(dataMap)

        const stationID = stationNameToID[selectedStation]
        updateAccuracyForStation(stationID ? dataMap[stationID] : undefined)

        const nextMessage = `Fetched statistics for ${payload.length} stations`
        if (lastFetchMessageRef.current !== nextMessage) {
          addLog(nextMessage)
          lastFetchMessageRef.current = nextMessage
        }
      } catch (error) {
        console.error('Error fetching statistics:', error)
        addLog('Error: Could not connect to backend statistics endpoint')
      } finally {
        clearTimeout(timeout)
      }
    }

    fetchStatistics()
    const interval = setInterval(fetchStatistics, Math.max(5000, POLL_MS))
    return () => clearInterval(interval)
  }, [API_URL, STATS_PATH, POLL_MS, REQUEST_TIMEOUT_MS, selectedStation])

  const currentStations = stationsByLine[selectedLine]

  useEffect(() => {
    try {
      window.localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify({ selectedLine, selectedStation }),
      )
    } catch {
      // Ignore storage write errors.
    }
  }, [selectedLine, selectedStation])

<<<<<<< ours
<<<<<<< ours
<<<<<<< ours
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

=======
>>>>>>> theirs
=======
>>>>>>> theirs
=======
>>>>>>> theirs
  const handleStationSelect = (station: string) => {
    setSelectedStation(station)

    const stationID = stationNameToID[station]

    if (stationID && stationData[stationID]) {
      const stats = stationData[stationID]
      updateAccuracyForStation(stats)
      addLog(`Station: ${station} | Inbound: ${stats.inbound_total} predictions | Outbound: ${stats.outbound_total} predictions`)
    } else {
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
          <div className="metrics-and-stations">
            <div className="accuracy-circles-container">
              <div className="accuracy-item">
                <div className={`percentage-circle ${inboundTotal > 0 ? (inboundAccuracy >= 70 ? 'good' : inboundAccuracy >= 50 ? 'moderate' : 'poor') : 'no-data'}`}>
                  <span className="percentage-number">
                    {inboundTotal > 0 ? `${inboundAccuracy}%` : 'N/A'}
                  </span>
                </div>
                <p className="accuracy-label">Inbound</p>
                <p className="direction-label">to Government Center</p>
                {inboundTotal > 0 ? <p className="prediction-count">({inboundTotal} predictions)</p> : null}
              </div>
              <div className="accuracy-item">
                <div className={`percentage-circle ${outboundTotal > 0 ? (outboundAccuracy >= 70 ? 'good' : outboundAccuracy >= 50 ? 'moderate' : 'poor') : 'no-data'}`}>
                  <span className="percentage-number">
                    {outboundTotal > 0 ? `${outboundAccuracy}%` : 'N/A'}
                  </span>
                </div>
                <p className="accuracy-label">Outbound</p>
                <p className="direction-label">to Boston College</p>
                {outboundTotal > 0 ? <p className="prediction-count">({outboundTotal} predictions)</p> : null}
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
              {supportedLines.map((line) => (
                <option key={line} value={line}>{line.replace('-', ' Line ')}</option>
              ))}
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
                      handleStationSelect(station.name)
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

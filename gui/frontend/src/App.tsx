import { useState, useEffect } from 'react'
import { Box, IconButton, Typography } from '@mui/material'
import { WindowLayout } from '@netlink/ui'
import SettingsIcon from '@mui/icons-material/Settings'
import PowerSettingsNewIcon from '@mui/icons-material/PowerSettingsNew'
import './App.css'
import { Connect, Disconnect, IsConnected, GetSettings } from "../wailsjs/go/main/App";
import { Settings } from './components/Settings'
import { PlugAnimation, PlugAnimationMode } from './components/PlugAnimation'

function App() {
  const [isConnected, setIsConnected] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [plugAnimationMode, setPlugAnimationMode] = useState<PlugAnimationMode | null>(null)
  const [errorMessage, setErrorMessage] = useState<string>('')
  const [serverName, setServerName] = useState('http://localhost:4535')
  const [deviceName, setDeviceName] = useState('netconnect-device')
  const [username, setUsername] = useState<string>('')
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false)
  const [isSettingsOpen, setIsSettingsOpen] = useState(false)

  // Fetch initial settings and connection status from Go backend on startup
  useEffect(() => {
    GetSettings()
      .then((settings) => {
        if (settings.serverAddress) setServerName(settings.serverAddress)
        if (settings.deviceName) setDeviceName(settings.deviceName)
        if (settings.username) setUsername(settings.username)
        if (typeof settings.isAuthenticated === 'boolean') {
          setIsAuthenticated(settings.isAuthenticated)
        }
      })
      .catch((err) => {
        console.error('[NetConnect] Failed to load initial settings:', err)
      })

    IsConnected()
      .then((status) => {
        setIsConnected(status)
        if (status) {
          setPlugAnimationMode('connected')
        }
      })
      .catch(() => {
        setIsConnected(false)
      })
  }, [])

  // Power button click handler: manages full Connect / Disconnect lifecycle
  const handlePowerClick = async () => {
    if (isLoading) return
    setIsLoading(true)
    setErrorMessage('')

    if (!isConnected) {
      try {
        console.log('[NetConnect Frontend] Initiating Connect...')
        await Connect()
        setIsConnected(true)
        setPlugAnimationMode('connect')
        console.log(`[NetConnect Frontend] Connected successfully to: ${serverName}`)
      } catch (err: any) {
        const msg = err?.message || String(err)
        console.error('[NetConnect Frontend] Connection failed:', msg)
        setErrorMessage(msg)
        setIsConnected(false)
      }
    } else {
      try {
        console.log('[NetConnect Frontend] Initiating Disconnect...')
        await Disconnect()
        setIsConnected(false)
        setPlugAnimationMode('disconnect')
        console.log(`[NetConnect Frontend] Disconnected from: ${serverName}`)
      } catch (err: any) {
        console.error('[NetConnect Frontend] Disconnect failed:', err)
      }
    }

    setIsLoading(false)
  }

  // Settings button click handler
  const handleSettingsClick = () => {
    setIsSettingsOpen(true)
    console.log('[NetConnect] Settings popup opened!')
  }

  const handleCloseSettings = () => {
    setIsSettingsOpen(false)
  }

  return (
    <WindowLayout padding={0}>
      <Box className="netconnect-frame">
        {/* Ambient background glows from NetLink design system */}
        <div className="bg-glow" />
        <div className="bg-glow-2" />

        {/* Plug Connection / Disconnection Background Layer */}
        {(isConnected || plugAnimationMode === 'disconnect') && (
          <PlugAnimation
            mode={plugAnimationMode || 'connected'}
            onComplete={() => {
              if (plugAnimationMode === 'connect') {
                setPlugAnimationMode('connected')
              } else if (plugAnimationMode === 'disconnect') {
                setPlugAnimationMode(null)
              }
            }}
          />
        )}

        {/* Top Header */}
        <Box className="netconnect-header">
          <IconButton
            className="settings-btn"
            onClick={handleSettingsClick}
            aria-label="Settings"
            disableRipple
          >
            <SettingsIcon className="settings-icon" />
          </IconButton>
          <Typography variant="h1" className="netconnect-title">
            NetConnect
          </Typography>
        </Box>

        {/* Center Power Button & Connection Status */}
        <Box className="netconnect-center">
          <button
            type="button"
            className={`power-button ${
              plugAnimationMode === 'disconnect'
                ? 'disconnecting'
                : isConnected
                ? 'connected'
                : 'disconnected'
            } ${isLoading ? 'loading' : ''}`}
            onClick={handlePowerClick}
            disabled={isLoading}
            aria-label={isConnected ? 'Disconnect' : 'Connect'}
          >
            <PowerSettingsNewIcon className="power-icon" />
          </button>

          <Typography className="status-text">
            {isLoading
              ? 'Connecting...'
              : isConnected
              ? `Connected to: ${serverName}`
              : `Disconnected from: ${serverName}`}
          </Typography>

          {errorMessage && (
            <Typography
              sx={{
                color: '#ff5252',
                fontSize: '0.8rem',
                mt: 1,
                px: 2,
                textAlign: 'center',
                maxWidth: '90%',
                wordBreak: 'break-word',
              }}
            >
              {errorMessage}
            </Typography>
          )}
        </Box>

        {/* Bottom space / User badge */}
        <Box className="netconnect-footer">
          {isAuthenticated && (
            <div className="user-badge-pill" onClick={handleSettingsClick} title="Authenticated with NetLink">
              <span className="user-badge-dot" />
              <span className="user-badge-text">{username || 'Linked'}</span>
            </div>
          )}
        </Box>

        {/* Settings Popup Modal */}
        <Settings
          isSettingsOpen={isSettingsOpen}
          handleCloseSettings={handleCloseSettings}
          serverName={serverName}
          setServerName={setServerName}
          deviceName={deviceName}
          setDeviceName={setDeviceName}
          username={username}
          setUsername={setUsername}
          isAuthenticated={isAuthenticated}
          setIsAuthenticated={setIsAuthenticated}
        />
      </Box>
    </WindowLayout>
  )
}

export default App

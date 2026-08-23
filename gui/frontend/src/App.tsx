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
  const [isErrorAnimation, setIsErrorAnimation] = useState(false)
  const [plugAnimationMode, setPlugAnimationMode] = useState<PlugAnimationMode | null>(null)
  const [errorMessage, setErrorMessage] = useState<string>('')
  const [serverName, setServerName] = useState('http://localhost:4535')
  const [deviceName, setDeviceName] = useState('netconnect-device')
  const [username, setUsername] = useState<string>('')
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false)
  const [isSettingsOpen, setIsSettingsOpen] = useState(false)

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
        console.error('[NetConnect] Failed to load settings:', err)
      })

    const checkConnection = () => {
      IsConnected()
        .then((status) => {
          setIsConnected((prev) => {
            if (prev && !status) {
              setPlugAnimationMode('disconnect')
            } else if (!prev && status) {
              setPlugAnimationMode('connected')
            }
            return status
          })
        })
        .catch(() => {
          setIsConnected((prev) => {
            if (prev) {
              setPlugAnimationMode('disconnect')
            }
            return false
          })
        })
    }

    checkConnection()
    const heartbeatInterval = setInterval(checkConnection, 1500)

    return () => clearInterval(heartbeatInterval)
  }, [])

  const handlePowerClick = async () => {
    if (isLoading) return
    setIsLoading(true)
    setErrorMessage('')
    setIsErrorAnimation(false)

    if (!isConnected) {
      if (!isAuthenticated) {
        setErrorMessage('Device is not authorized with NetLink. Please log in first.')
        setIsConnected(false)
        setIsErrorAnimation(true)
        setPlugAnimationMode('error')
        setTimeout(() => setIsErrorAnimation(false), 1600)
        setIsLoading(false)
        return
      }

      try {
        await Connect()
        setIsConnected(true)
        setPlugAnimationMode('connect')
      } catch (err: any) {
        const msg = err?.message || String(err)
        setErrorMessage(msg)
        setIsConnected(false)
        setIsErrorAnimation(true)
        setPlugAnimationMode('error')
        setTimeout(() => setIsErrorAnimation(false), 1600)
      }
    } else {
      try {
        await Disconnect()
        setIsConnected(false)
        setPlugAnimationMode('disconnect')
      } catch (err: any) {
        console.error('[NetConnect] Disconnect failed:', err)
      }
    }

    setIsLoading(false)
  }

  const handleSettingsClick = () => {
    setIsSettingsOpen(true)
  }

  const handleCloseSettings = () => {
    setIsSettingsOpen(false)
  }

  return (
    <WindowLayout padding={0}>
      <Box className="netconnect-frame">
        <div className="bg-glow" />
        <div className="bg-glow-2" />

        {(isConnected || plugAnimationMode !== null) && (
          <PlugAnimation
            mode={plugAnimationMode || 'connected'}
            onComplete={() => {
              if (plugAnimationMode === 'connect') {
                setPlugAnimationMode('connected')
              } else if (plugAnimationMode === 'disconnect' || plugAnimationMode === 'error') {
                setPlugAnimationMode(null)
              }
            }}
          />
        )}


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

        <Box className="netconnect-center">
          <button
            type="button"
            className={`power-button ${
              plugAnimationMode === 'disconnect'
                ? 'disconnecting'
                : isConnected
                ? 'connected'
                : 'disconnected'
            } ${isLoading ? 'loading' : ''} ${isErrorAnimation ? 'error-pulse' : ''}`}
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

        <Box className="netconnect-footer">
          {isAuthenticated && (
            <div className="user-badge-pill" onClick={handleSettingsClick} title="Authenticated with NetLink">
              <span className="user-badge-dot" />
              <span className="user-badge-text">{username || 'Linked'}</span>
            </div>
          )}
        </Box>

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

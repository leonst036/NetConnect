import { useState } from 'react'
import { Box, IconButton, Typography } from '@mui/material'
import { WindowLayout } from '@netlink/ui'
import SettingsIcon from '@mui/icons-material/Settings'
import PowerSettingsNewIcon from '@mui/icons-material/PowerSettingsNew'
import CloseIcon from '@mui/icons-material/Close'
import './App.css'
import { Connect } from "../wailsjs/go/main/App";

function App() {
  const [isConnected, setIsConnected] = useState(true)
  const [serverName, setServerName] = useState('<server>')
  const [isSettingsOpen, setIsSettingsOpen] = useState(false)

  // Power button click handler with console log
  const handlePowerClick = () => {
    const nextState = !isConnected
    setIsConnected(nextState)
    if (nextState) {
      Connect()
    }
    console.log(
      `[NetConnect] Power button clicked! Status: ${nextState ? 'Connected' : 'Disconnected'} (Server: ${serverName})`
    )
  }

  // Settings button click handler with console log
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
            className={`power-button ${isConnected ? 'connected' : 'disconnected'}`}
            onClick={handlePowerClick}
            aria-label={isConnected ? 'Disconnect' : 'Connect'}
          >
            <PowerSettingsNewIcon className="power-icon" />
          </button>

          <Typography className="status-text">
            {isConnected ? `Connected to: ${serverName}` : `Disconnected from: ${serverName}`}
          </Typography>
        </Box>

        {/* Bottom space for visual balance */}
        <Box className="netconnect-footer" />

        {/* Settings Popup Modal with fixed dimensions */}
        {isSettingsOpen && (
          <div className="popup-overlay" onClick={handleCloseSettings}>
            <div
              className="popup-window"
              onClick={(e) => e.stopPropagation()}
            >
              <div className="popup-header">
                <Typography variant="h6" className="popup-title">
                  Settings
                </Typography>
                <IconButton
                  className="popup-close-btn"
                  onClick={handleCloseSettings}
                  size="small"
                >
                  <CloseIcon fontSize="small" />
                </IconButton>
              </div>

              <div className="popup-body">
                <label className="popup-label">Server Address</label>
                <input
                  type="text"
                  className="popup-input"
                  value={serverName}
                  onChange={(e) => setServerName(e.target.value)}
                  placeholder="e.g. 192.168.1.100"
                />

                <div className="popup-info">
                  Hyprland popup window configuration active.
                </div>
              </div>

              <div className="popup-footer">
                <button
                  type="button"
                  className="popup-action-btn"
                  onClick={handleCloseSettings}
                >
                  Save & Close
                </button>
              </div>
            </div>
          </div>
        )}
      </Box>
    </WindowLayout>
  )
}

export default App

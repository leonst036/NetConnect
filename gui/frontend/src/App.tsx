import { useState } from 'react'
import { Box, IconButton, Typography } from '@mui/material'
import { WindowLayout } from '@netlink/ui'
import SettingsIcon from '@mui/icons-material/Settings'
import PowerSettingsNewIcon from '@mui/icons-material/PowerSettingsNew'
import './App.css'
import { Connect } from "../wailsjs/go/main/App";
import { Settings } from './components/Settings'

function App() {
  const [isConnected, setIsConnected] = useState(true)
  const [serverName, setServerName] = useState('<server>')
  const [deviceName, setDeviceName] = useState('<device>')
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

        {/* Settings Popup Modal */}
        <Settings
          isSettingsOpen={isSettingsOpen}
          handleCloseSettings={handleCloseSettings}
          serverName={serverName}
          setServerName={setServerName}
          deviceName={deviceName}
          setDeviceName={setDeviceName}
        />
      </Box>
    </WindowLayout>
  )
}

export default App

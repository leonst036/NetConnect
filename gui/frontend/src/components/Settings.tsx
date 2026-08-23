import { IconButton, Typography } from "@mui/material";
import CloseIcon from '@mui/icons-material/Close';

interface SettingsProps {
    isSettingsOpen: boolean;
    handleCloseSettings: () => void;
    serverName: string;
    setServerName: (serverName: string) => void;
    deviceName: string;
    setDeviceName: (deviceName: string) => void;
}

export const Settings = ({ isSettingsOpen, handleCloseSettings, serverName, setServerName, deviceName, setDeviceName }: SettingsProps) => {
    return (
        <>  {isSettingsOpen && (
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
                            placeholder="e.g. netlink.example.com"
                        />
                        <label className="popup-label">Device Name</label>
                        <input
                            type="text"
                            className="popup-input"
                            value={deviceName}
                            onChange={(e) => setDeviceName(e.target.value)}
                            placeholder="e.g. My Device"
                        />
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
        )}</>
    )
}
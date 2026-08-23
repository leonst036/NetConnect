import { useState, useEffect } from "react";
import { IconButton, Typography } from "@mui/material";
import CloseIcon from '@mui/icons-material/Close';
import { SaveSettings } from "../../wailsjs/go/main/App";

interface SettingsProps {
    isSettingsOpen: boolean;
    handleCloseSettings: () => void;
    serverName: string;
    setServerName: (serverName: string) => void;
    deviceName: string;
    setDeviceName: (deviceName: string) => void;
}

export const Settings = ({
    isSettingsOpen,
    handleCloseSettings,
    serverName,
    setServerName,
    deviceName,
    setDeviceName
}: SettingsProps) => {
    const [tempServerName, setTempServerName] = useState(serverName);
    const [tempDeviceName, setTempDeviceName] = useState(deviceName);
    const [isSaving, setIsSaving] = useState(false);
    const [errorMessage, setErrorMessage] = useState<string | null>(null);

    // Sync form inputs when modal opens or props update
    useEffect(() => {
        if (isSettingsOpen) {
            setTempServerName(serverName);
            setTempDeviceName(deviceName);
            setErrorMessage(null);
            setIsSaving(false);
        }
    }, [isSettingsOpen, serverName, deviceName]);

    const handleSave = async () => {
        setIsSaving(true);
        setErrorMessage(null);

        try {
            await SaveSettings(tempServerName, tempDeviceName);
            setServerName(tempServerName);
            setDeviceName(tempDeviceName);
            handleCloseSettings();
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : String(err);
            setErrorMessage(message || "Failed to update settings and validate device token");
        } finally {
            setIsSaving(false);
        }
    };

    return (
        <>
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
                                disabled={isSaving}
                            >
                                <CloseIcon fontSize="small" />
                            </IconButton>
                        </div>

                        <div className="popup-body">
                            <label className="popup-label">Server Address</label>
                            <input
                                type="text"
                                className="popup-input"
                                value={tempServerName}
                                onChange={(e) => setTempServerName(e.target.value)}
                                placeholder="e.g. http://localhost:5173"
                                disabled={isSaving}
                            />
                            <label className="popup-label">Device Name</label>
                            <input
                                type="text"
                                className="popup-input"
                                value={tempDeviceName}
                                onChange={(e) => setTempDeviceName(e.target.value)}
                                placeholder="e.g. My Device"
                                disabled={isSaving}
                            />

                            {errorMessage && (
                                <div className="popup-error">
                                    {errorMessage}
                                </div>
                            )}
                        </div>

                        <div className="popup-footer">
                            <button
                                type="button"
                                className="popup-action-btn"
                                onClick={handleSave}
                                disabled={isSaving}
                            >
                                {isSaving ? "Validating & Saving..." : "Save & Close"}
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </>
    );
};
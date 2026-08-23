import { useState, useEffect, useRef } from "react";
import { IconButton, Typography, CircularProgress } from "@mui/material";
import CloseIcon from '@mui/icons-material/Close';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import LockOpenIcon from '@mui/icons-material/LockOpen';
import LogoutIcon from '@mui/icons-material/Logout';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import { 
  SaveSettings, 
  StartDeviceLogin, 
  PollDeviceLogin, 
  Logout, 
  OpenVerificationURL 
} from "../../wailsjs/go/main/App";

interface SettingsProps {
  isSettingsOpen: boolean;
  handleCloseSettings: () => void;
  serverName: string;
  setServerName: (serverName: string) => void;
  deviceName: string;
  setDeviceName: (deviceName: string) => void;
  username?: string;
  setUsername: (username: string) => void;
  isAuthenticated: boolean;
  setIsAuthenticated: (auth: boolean) => void;
}

export const Settings = ({
  isSettingsOpen,
  handleCloseSettings,
  serverName,
  setServerName,
  deviceName,
  setDeviceName,
  username,
  setUsername,
  isAuthenticated,
  setIsAuthenticated,
}: SettingsProps) => {
  const [tempServerName, setTempServerName] = useState(serverName);
  const [tempDeviceName, setTempDeviceName] = useState(deviceName);
  const [isSaving, setIsSaving] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  // Device pairing flow state
  const [isPairing, setIsPairing] = useState(false);
  const [userCode, setUserCode] = useState<string>('');
  const [verificationUrl, setVerificationUrl] = useState<string>('');
  const [pairingStatus, setPairingStatus] = useState<string>('');
  const [copied, setCopied] = useState(false);
  const pollTimerRef = useRef<number | null>(null);
  const isCancelledRef = useRef(false);

  // Clean up polling timer
  const stopPolling = () => {
    isCancelledRef.current = true;
    if (pollTimerRef.current) {
      clearTimeout(pollTimerRef.current);
      pollTimerRef.current = null;
    }
  };

  // Sync inputs on open
  useEffect(() => {
    if (isSettingsOpen) {
      setTempServerName(serverName);
      setTempDeviceName(deviceName);
      setErrorMessage(null);
      setIsSaving(false);
      setIsPairing(false);
      setUserCode('');
      setVerificationUrl('');
      setCopied(false);
      stopPolling();
    } else {
      stopPolling();
    }
    return () => stopPolling();
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
      setErrorMessage(message || "Failed to update settings");
    } finally {
      setIsSaving(false);
    }
  };

  const handleStartDeviceLogin = async () => {
    setIsSaving(true);
    setErrorMessage(null);
    setIsPairing(true);
    setUserCode('');
    setVerificationUrl('');
    setPairingStatus('Requesting authorization code...');
    isCancelledRef.current = false;

    try {
      const res = await StartDeviceLogin(tempServerName, tempDeviceName);
      if (res.error) {
        throw new Error(res.error);
      }

      setUserCode(res.user_code);
      const openUrl = res.verification_uri_complete || res.verification_uri;
      setVerificationUrl(openUrl);
      setPairingStatus('Waiting for authorization in browser...');

      // Open browser automatically
      if (openUrl) {
        try {
          OpenVerificationURL(openUrl);
        } catch (e) {
          console.warn('[NetConnect] Could not open browser automatically:', e);
        }
      }

      // Start polling loop
      const intervalSec = res.interval > 0 ? res.interval : 2;
      const deviceCode = res.device_code;

      const pollLoop = async () => {
        if (isCancelledRef.current) return;

        try {
          const pollRes = await PollDeviceLogin(deviceCode);
          if (pollRes.status === 'approved' || (pollRes.token && !pollRes.error)) {
            setPairingStatus('Authorized successfully!');
            setIsAuthenticated(true);
            setErrorMessage(null);
            if (pollRes.username) setUsername(pollRes.username);
            if (pollRes.target_id) {
              setTempDeviceName(pollRes.target_id);
              setDeviceName(pollRes.target_id);
            }
            setServerName(tempServerName);
            setTimeout(() => {
              setIsPairing(false);
            }, 1200);
            return;
          } else if (pollRes.status === 'expired') {
            setErrorMessage('Authorization code expired. Please try again.');
            setIsPairing(false);
            return;
          } else if (pollRes.status === 'denied') {
            setErrorMessage('Authorization was denied in the web browser.');
            setIsPairing(false);
            return;
          }
        } catch (pollErr: unknown) {
          // Keep polling unless explicit cancel
          console.debug('[NetConnect] Poll tick:', pollErr);
        }

        if (!isCancelledRef.current) {
          pollTimerRef.current = window.setTimeout(pollLoop, intervalSec * 1000);
        }
      };

      pollTimerRef.current = window.setTimeout(pollLoop, intervalSec * 1000);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      setErrorMessage(message || "Failed to start device login flow");
      setIsPairing(false);
    } finally {
      setIsSaving(false);
    }
  };

  const handleCancelPairing = () => {
    stopPolling();
    setIsPairing(false);
    setUserCode('');
    setVerificationUrl('');
  };

  const handleLogout = async () => {
    try {
      await Logout();
      setIsAuthenticated(false);
      setUsername('');
    } catch (err: unknown) {
      console.error('[NetConnect] Logout error:', err);
    }
  };

  const handleCopyCode = () => {
    if (userCode) {
      navigator.clipboard.writeText(userCode);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <>
      {isSettingsOpen && (
        <div className="popup-overlay" onClick={handleCloseSettings}>
          <div className="popup-window" onClick={(e) => e.stopPropagation()}>
            <div className="popup-header">
              <Typography variant="h6" className="popup-title">
                Settings & Auth
              </Typography>
              <IconButton
                className="popup-close-btn"
                onClick={handleCloseSettings}
                size="small"
                disabled={isSaving && !isPairing}
              >
                <CloseIcon fontSize="small" />
              </IconButton>
            </div>

            <div className="popup-body">
              {/* Account / User Section */}
              <div className="auth-card">
                <div className="auth-card-header">
                  <span className="auth-card-title">NetLink Authentication</span>
                  {isAuthenticated ? (
                    <span className="auth-badge-connected">
                      <CheckCircleIcon sx={{ fontSize: 14 }} /> Connected
                    </span>
                  ) : (
                    <span className="auth-badge-unlinked">
                      <LockOpenIcon sx={{ fontSize: 14 }} /> Not Linked
                    </span>
                  )}
                </div>

                {isAuthenticated ? (
                  <div className="auth-card-content">
                    <div className="auth-user-info">
                      <span className="auth-label">User Account:</span>
                      <span className="auth-value">{username || 'Authorized Device'}</span>
                    </div>
                    <button
                      type="button"
                      className="auth-logout-btn"
                      onClick={handleLogout}
                    >
                      <LogoutIcon sx={{ fontSize: 14 }} /> Log Out
                    </button>
                  </div>
                ) : isPairing ? (
                  <div className="auth-pairing-box">
                    <div className="pairing-spinner-row">
                      <CircularProgress size={18} sx={{ color: '#d8b4fe' }} />
                      <span className="pairing-status-text">{pairingStatus}</span>
                    </div>

                    {userCode && (
                      <div className="pairing-code-container">
                        <div className="pairing-code-label">Confirmation Code</div>
                        <div className="pairing-code-value" onClick={handleCopyCode} title="Click to copy">
                          {userCode}
                          <IconButton size="small" sx={{ color: '#d8b4fe', ml: 1 }}>
                            <ContentCopyIcon sx={{ fontSize: 16 }} />
                          </IconButton>
                        </div>
                        {copied && <span className="copied-hint">Copied to clipboard!</span>}
                      </div>
                    )}

                    <div className="pairing-actions">
                      {verificationUrl && (
                        <button
                          type="button"
                          className="auth-browser-btn"
                          onClick={() => OpenVerificationURL(verificationUrl)}
                        >
                          <OpenInNewIcon sx={{ fontSize: 14 }} /> Open in Browser
                        </button>
                      )}
                      <button
                        type="button"
                        className="auth-cancel-btn"
                        onClick={handleCancelPairing}
                      >
                        Cancel
                      </button>
                    </div>
                  </div>
                ) : (
                  <div className="auth-card-content">
                    <span className="auth-hint">
                      Log in to your NetLink web account to select and authorize your target servers.
                    </span>
                    <button
                      type="button"
                      className="auth-login-btn"
                      onClick={handleStartDeviceLogin}
                      disabled={isSaving}
                    >
                      <OpenInNewIcon sx={{ fontSize: 14 }} /> Log In with NetLink
                    </button>
                  </div>
                )}
              </div>

              {/* Server & Device Fields */}
              <label className="popup-label">Relay Server URL</label>
              <input
                type="text"
                className="popup-input"
                value={tempServerName}
                onChange={(e) => setTempServerName(e.target.value)}
                placeholder="e.g. https://relay.example.com"
                disabled={isSaving || isPairing}
              />

              <label className="popup-label">Device Identifier</label>
              <input
                type="text"
                className="popup-input"
                value={tempDeviceName}
                onChange={(e) => setTempDeviceName(e.target.value)}
                placeholder="e.g. netconnect-laptop"
                disabled={isSaving || isPairing}
              />

              {errorMessage && <div className="popup-error">{errorMessage}</div>}
            </div>

            <div className="popup-footer">
              <button
                type="button"
                className="popup-action-btn"
                onClick={handleSave}
                disabled={isSaving || isPairing}
              >
                {isSaving ? "Saving..." : "Save & Close"}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
};
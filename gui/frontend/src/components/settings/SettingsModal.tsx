import { useState, useEffect, useRef } from "react";
import { 
  Alert, 
  Box,
  Button, 
  Dialog, 
  DialogActions, 
  DialogContent, 
  DialogTitle, 
  IconButton, 
  Switch,
  Typography 
} from "@mui/material";
import CloseRoundedIcon from '@mui/icons-material/CloseRounded';
import { 
  SaveSettings, 
  StartDeviceLogin, 
  PollDeviceLogin, 
  Logout, 
  OpenVerificationURL,
  GetAutoStart,
  SetAutoStart
} from "../../../wailsjs/go/main/App";
import { AuthCard } from "./AuthCard";
import { ServerConfigFields } from "./ServerConfigFields";


export interface SettingsModalProps {
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

export const SettingsModal = ({
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
}: SettingsModalProps) => {
  const [tempServerName, setTempServerName] = useState(serverName);
  const [tempDeviceName, setTempDeviceName] = useState(deviceName);
  const [isSaving, setIsSaving] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const [isPairing, setIsPairing] = useState(false);
  const [userCode, setUserCode] = useState('');
  const [verificationUrl, setVerificationUrl] = useState('');
  const [pairingStatus, setPairingStatus] = useState('');
  const [copied, setCopied] = useState(false);
  const [autoStart, setAutoStart] = useState(false);
  const [isTogglingAutoStart, setIsTogglingAutoStart] = useState(false);
  const pollTimerRef = useRef<number | null>(null);
  const isCancelledRef = useRef(false);

  const stopPolling = () => {
    isCancelledRef.current = true;
    if (pollTimerRef.current) {
      clearTimeout(pollTimerRef.current);
      pollTimerRef.current = null;
    }
  };

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
      GetAutoStart().then((res) => setAutoStart(res)).catch(() => {});
    } else {
      stopPolling();
    }
    return () => stopPolling();
  }, [isSettingsOpen, serverName, deviceName]);

  const handleToggleAutoStart = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const checked = event.target.checked;
    setIsTogglingAutoStart(true);
    setAutoStart(checked);
    try {
      await SetAutoStart(checked);
    } catch (err: unknown) {
      setAutoStart(!checked);
      const message = err instanceof Error ? err.message : String(err);
      setErrorMessage(message || "Failed to update auto-start setting");
    } finally {
      setIsTogglingAutoStart(false);
    }
  };


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

      if (openUrl) {
        try {
          OpenVerificationURL(openUrl);
        } catch {
          // ignore browser open error
        }
      }

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
        } catch {
          // continue polling
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
    } catch {
      // ignore
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
    <Dialog
      open={isSettingsOpen}
      onClose={isSaving && !isPairing ? undefined : handleCloseSettings}
      maxWidth="xs"
      fullWidth
      PaperProps={{
        sx: {
          borderRadius: '24px',
          bgcolor: 'background.paper',
          backgroundImage: 'none',
          border: '1px solid',
          borderColor: 'divider',
          p: 1,
        },
      }}
    >
      <DialogTitle sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', pb: 1 }}>
        <Typography variant="h6" fontWeight={600} component="div">
          Settings & Auth
        </Typography>
        <IconButton
          onClick={handleCloseSettings}
          size="small"
          disabled={isSaving && !isPairing}
          sx={{ color: 'text.secondary' }}
        >
          <CloseRoundedIcon fontSize="small" />
        </IconButton>
      </DialogTitle>

      <DialogContent sx={{ px: 2, py: 1 }}>
        <AuthCard
          isAuthenticated={isAuthenticated}
          username={username}
          isPairing={isPairing}
          isSaving={isSaving}
          userCode={userCode}
          verificationUrl={verificationUrl}
          pairingStatus={pairingStatus}
          copied={copied}
          onStartLogin={handleStartDeviceLogin}
          onLogout={handleLogout}
          onCancelPairing={handleCancelPairing}
          onCopyCode={handleCopyCode}
        />

        <ServerConfigFields
          serverAddress={tempServerName}
          onServerAddressChange={setTempServerName}
          deviceName={tempDeviceName}
          onDeviceNameChange={setTempDeviceName}
          disabled={isSaving || isPairing}
        />

        {/* Auto-Start Systemd Setting */}
        <Box
          sx={{
            mt: 2,
            p: 1.5,
            borderRadius: '16px',
            bgcolor: 'action.hover',
            border: '1px solid',
            borderColor: 'divider',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <Box sx={{ pr: 1 }}>
            <Typography variant="body2" fontWeight={600} color="text.primary">
              Start on Boot
            </Typography>
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.25 }}>
              Automatically launch NetConnect systemd service on boot
            </Typography>
          </Box>
          <Switch
            checked={autoStart}
            onChange={handleToggleAutoStart}
            disabled={isSaving || isPairing || isTogglingAutoStart}
            color="primary"
          />
        </Box>

        {errorMessage && (
          <Alert severity="error" sx={{ mt: 2, borderRadius: '12px' }}>
            {errorMessage}
          </Alert>
        )}

      </DialogContent>

      <DialogActions sx={{ px: 2, pb: 1.5, pt: 1, gap: 1 }}>
        <Button
          onClick={handleCloseSettings}
          color="inherit"
          disabled={isSaving && !isPairing}
          sx={{ borderRadius: '20px', textTransform: 'none' }}
        >
          Cancel
        </Button>
        <Button
          variant="contained"
          onClick={handleSave}
          disabled={isSaving || isPairing}
          sx={{
            borderRadius: '20px',
            textTransform: 'none',
            fontWeight: 600,
            px: 3,
          }}
        >
          {isSaving ? "Saving..." : "Save & Close"}
        </Button>
      </DialogActions>
    </Dialog>
  );
};


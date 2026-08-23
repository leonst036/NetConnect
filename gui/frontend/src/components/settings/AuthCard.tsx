import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import LockOpenIcon from '@mui/icons-material/LockOpen';
import LogoutIcon from '@mui/icons-material/Logout';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import { DevicePairingBox } from './DevicePairingBox';

interface AuthCardProps {
  isAuthenticated: boolean;
  username?: string;
  isPairing: boolean;
  isSaving: boolean;
  userCode: string;
  verificationUrl: string;
  pairingStatus: string;
  copied: boolean;
  onStartLogin: () => void;
  onLogout: () => void;
  onCancelPairing: () => void;
  onCopyCode: () => void;
}

export const AuthCard = ({
  isAuthenticated,
  username,
  isPairing,
  isSaving,
  userCode,
  verificationUrl,
  pairingStatus,
  copied,
  onStartLogin,
  onLogout,
  onCancelPairing,
  onCopyCode,
}: AuthCardProps) => {
  return (
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
            onClick={onLogout}
          >
            <LogoutIcon sx={{ fontSize: 14 }} /> Log Out
          </button>
        </div>
      ) : isPairing ? (
        <DevicePairingBox
          userCode={userCode}
          verificationUrl={verificationUrl}
          pairingStatus={pairingStatus}
          copied={copied}
          onCopyCode={onCopyCode}
          onCancel={onCancelPairing}
        />
      ) : (
        <div className="auth-card-content">
          <span className="auth-hint">
            Log in to your NetLink web account to select and authorize your target servers.
          </span>
          <button
            type="button"
            className="auth-login-btn"
            onClick={onStartLogin}
            disabled={isSaving}
          >
            <OpenInNewIcon sx={{ fontSize: 14 }} /> Log In with NetLink
          </button>
        </div>
      )}
    </div>
  );
};

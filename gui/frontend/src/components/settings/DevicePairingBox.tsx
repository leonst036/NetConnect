import { IconButton, CircularProgress } from "@mui/material";
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import { OpenVerificationURL } from "../../../wailsjs/go/main/App";

interface DevicePairingBoxProps {
  userCode: string;
  verificationUrl: string;
  pairingStatus: string;
  copied: boolean;
  onCopyCode: () => void;
  onCancel: () => void;
}

export const DevicePairingBox = ({
  userCode,
  verificationUrl,
  pairingStatus,
  copied,
  onCopyCode,
  onCancel,
}: DevicePairingBoxProps) => {
  return (
    <div className="auth-pairing-box">
      <div className="pairing-spinner-row">
        <CircularProgress size={18} sx={{ color: '#d8b4fe' }} />
        <span className="pairing-status-text">{pairingStatus}</span>
      </div>

      {userCode && (
        <div className="pairing-code-container">
          <div className="pairing-code-label">Confirmation Code</div>
          <div className="pairing-code-value" onClick={onCopyCode} title="Click to copy">
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
          onClick={onCancel}
        >
          Cancel
        </button>
      </div>
    </div>
  );
};

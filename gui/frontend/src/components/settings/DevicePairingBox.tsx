import { Box, Button, CircularProgress, IconButton, Stack, Tooltip, Typography } from "@mui/material";
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import CheckIcon from '@mui/icons-material/Check';
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
    <Stack spacing={2} sx={{ width: '100%', pt: 0.5 }}>
      <Stack direction="row" spacing={1.5} alignItems="center">
        <CircularProgress size={18} />
        <Typography variant="body2" color="text.secondary">
          {pairingStatus}
        </Typography>
      </Stack>

      {userCode && (
        <Box
          sx={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            p: 1.5,
            borderRadius: '16px',
            bgcolor: 'action.hover',
            border: '1px dashed',
            borderColor: 'divider',
            gap: 0.5,
          }}
        >
          <Typography variant="caption" color="text.secondary" sx={{ letterSpacing: 0.5, textTransform: 'uppercase' }}>
            Confirmation Code
          </Typography>
          <Stack direction="row" alignItems="center" spacing={1}>
            <Typography
              variant="h5"
              sx={{
                fontFamily: 'monospace',
                fontWeight: 700,
                letterSpacing: 3,
                cursor: 'pointer',
              }}
              onClick={onCopyCode}
            >
              {userCode}
            </Typography>
            <Tooltip title={copied ? "Copied!" : "Copy code"} arrow>
              <IconButton size="small" onClick={onCopyCode} color={copied ? "success" : "default"}>
                {copied ? <CheckIcon fontSize="small" /> : <ContentCopyIcon fontSize="small" />}
              </IconButton>
            </Tooltip>
          </Stack>
        </Box>
      )}

      <Stack direction="row" spacing={1} justifyContent="flex-end" sx={{ pt: 0.5 }}>
        <Button
          variant="text"
          color="inherit"
          onClick={onCancel}
          sx={{ borderRadius: '20px', textTransform: 'none' }}
        >
          Cancel
        </Button>
        {verificationUrl && (
          <Button
            variant="contained"
            startIcon={<OpenInNewIcon fontSize="small" />}
            onClick={() => OpenVerificationURL(verificationUrl)}
            sx={{ borderRadius: '20px', textTransform: 'none', fontWeight: 600 }}
          >
            Open in Browser
          </Button>
        )}
      </Stack>
    </Stack>
  );
};


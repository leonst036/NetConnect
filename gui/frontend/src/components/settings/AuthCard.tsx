import { Box, Button, Card, CardContent, Chip, Stack, Typography } from '@mui/material';
import CheckCircleRoundedIcon from '@mui/icons-material/CheckCircleRounded';
import LockOpenRoundedIcon from '@mui/icons-material/LockOpenRounded';
import LogoutRoundedIcon from '@mui/icons-material/LogoutRounded';
import OpenInNewRoundedIcon from '@mui/icons-material/OpenInNewRounded';
import AccountCircleOutlinedIcon from '@mui/icons-material/AccountCircleOutlined';
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
    <Card
      variant="outlined"
      sx={{
        borderRadius: '16px',
        bgcolor: 'background.paper',
        borderColor: 'divider',
        mb: 2,
      }}
    >
      <CardContent sx={{ p: 2, '&:last-child': { pb: 2 } }}>
        <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ mb: 1.5 }}>
          <Typography variant="subtitle2" fontWeight={600} color="text.primary">
            NetLink Authentication
          </Typography>
          {isAuthenticated ? (
            <Chip
              icon={<CheckCircleRoundedIcon />}
              label="Connected"
              size="small"
              color="success"
              variant="outlined"
              sx={{ borderRadius: '8px', fontWeight: 500 }}
            />
          ) : (
            <Chip
              icon={<LockOpenRoundedIcon />}
              label="Not Linked"
              size="small"
              variant="outlined"
              sx={{ borderRadius: '8px', color: 'text.secondary', borderColor: 'divider' }}
            />
          )}
        </Stack>

        {isAuthenticated ? (
          <Stack spacing={1.5}>
            <Box
              sx={{
                display: 'flex',
                alignItems: 'center',
                gap: 1.5,
                p: 1.25,
                borderRadius: '12px',
                bgcolor: 'action.hover',
              }}
            >
              <AccountCircleOutlinedIcon sx={{ color: 'primary.main', fontSize: 24 }} />
              <Box sx={{ minWidth: 0, flex: 1 }}>
                <Typography variant="caption" color="text.secondary" display="block" lineHeight={1.2}>
                  Signed in as
                </Typography>
                <Typography variant="body2" fontWeight={600} noWrap>
                  {username || 'Authorized Device'}
                </Typography>
              </Box>
            </Box>
            <Button
              variant="outlined"
              color="error"
              size="small"
              startIcon={<LogoutRoundedIcon fontSize="small" />}
              onClick={onLogout}
              sx={{ borderRadius: '20px', textTransform: 'none', alignSelf: 'flex-start' }}
            >
              Log Out
            </Button>
          </Stack>
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
          <Stack spacing={2}>
            <Typography variant="body2" color="text.secondary" lineHeight={1.4}>
              Log in to your NetLink web account to select and authorize your target servers.
            </Typography>
            <Button
              variant="contained"
              fullWidth
              startIcon={<OpenInNewRoundedIcon fontSize="small" />}
              onClick={onStartLogin}
              disabled={isSaving}
              sx={{
                borderRadius: '20px',
                textTransform: 'none',
                fontWeight: 600,
                py: 1,
              }}
            >
              Log In with NetLink
            </Button>
          </Stack>
        )}
      </CardContent>
    </Card>
  );
};


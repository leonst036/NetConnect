import { TextField, Stack, InputAdornment } from '@mui/material';
import DnsOutlinedIcon from '@mui/icons-material/DnsOutlined';
import DevicesOutlinedIcon from '@mui/icons-material/DevicesOutlined';

interface ServerConfigFieldsProps {
  serverAddress: string;
  onServerAddressChange: (val: string) => void;
  deviceName: string;
  onDeviceNameChange: (val: string) => void;
  disabled: boolean;
}

export const ServerConfigFields = ({
  serverAddress,
  onServerAddressChange,
  deviceName,
  onDeviceNameChange,
  disabled,
}: ServerConfigFieldsProps) => {
  return (
    <Stack spacing={2} sx={{ mt: 1 }}>
      <TextField
        label="Relay Server URL"
        placeholder="e.g. localhost:5171"
        value={serverAddress}
        onChange={(e) => onServerAddressChange(e.target.value)}
        disabled={disabled}
        fullWidth
        size="small"
        variant="outlined"
        InputProps={{
          startAdornment: (
            <InputAdornment position="start">
              <DnsOutlinedIcon fontSize="small" sx={{ color: 'text.secondary' }} />
            </InputAdornment>
          ),
          sx: { borderRadius: '12px' },
        }}
      />
      <TextField
        label="Device Identifier"
        placeholder="e.g. netconnect-device"
        value={deviceName}
        onChange={(e) => onDeviceNameChange(e.target.value)}
        disabled={disabled}
        fullWidth
        size="small"
        variant="outlined"
        InputProps={{
          startAdornment: (
            <InputAdornment position="start">
              <DevicesOutlinedIcon fontSize="small" sx={{ color: 'text.secondary' }} />
            </InputAdornment>
          ),
          sx: { borderRadius: '12px' },
        }}
      />
    </Stack>
  );
};


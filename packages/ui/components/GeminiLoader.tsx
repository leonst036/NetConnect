import { Box } from '@mui/material';
import './GeminiLoader.css';

export function GeminiLoader({ size = 48 }: { size?: number }) {
  return (
    <Box className="loader-container" style={{ width: size, height: size }}>
      <div className="gemini-blob"></div>
      <div className="gemini-core"></div>
    </Box>
  );
}

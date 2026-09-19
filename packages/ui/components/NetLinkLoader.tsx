import { Box } from '@mui/material';
import './NetLinkLoader.css';

export interface NetLinkLoaderProps {
  size?: number;
  className?: string;
}

export function NetLinkLoader({ size = 48, className = '' }: NetLinkLoaderProps) {
  return (
    <Box
      className={`loader-container netlink-loader ${className}`}
      role="status"
      aria-label="Loading"
      sx={{ width: size, height: size }}
    >
      <svg
        className="netlink-spinner-svg"
        viewBox="0 0 48 48"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
      >
        <circle
          className="netlink-spinner-track"
          cx="24"
          cy="24"
          r="20"
          strokeWidth="2.5"
        />
        <circle
          className="netlink-spinner-outer"
          cx="24"
          cy="24"
          r="20"
          strokeWidth="2.5"
          strokeLinecap="round"
        />
        <circle
          className="netlink-spinner-inner"
          cx="24"
          cy="24"
          r="12"
          strokeWidth="2"
          strokeLinecap="round"
        />
        <circle
          className="netlink-spinner-node"
          cx="24"
          cy="24"
          r="3"
        />
      </svg>
    </Box>
  );
}
export default NetLinkLoader;

import React from 'react';
import { ThemeProvider, CssBaseline, Box } from '@mui/material';
import { getAppTheme } from '../index';
import '../global.css';

interface WindowLayoutProps {
  children: React.ReactNode;
  themeName?: string;
  padding?: string | number;
}

export function WindowLayout({ children, themeName = 'Dark', padding = '20px' }: WindowLayoutProps) {
  const theme = getAppTheme(themeName);
  
  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <Box sx={{ 
        width: '100%', 
        height: '100vh', 
        display: 'flex', 
        flexDirection: 'column', 
        background: theme.palette.background.default,
        color: theme.palette.text.primary,
        overflow: 'hidden'
      }}>
        <Box sx={{ flexGrow: 1, padding, overflowY: 'auto', display: 'flex', flexDirection: 'column' }}>
          {children}
        </Box>
      </Box>
    </ThemeProvider>
  );
}

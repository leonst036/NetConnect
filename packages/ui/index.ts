import { createTheme } from '@mui/material/styles';

export const getAppTheme = (themeName: string) => {
    const isDark = themeName.toLowerCase() !== 'light';
    return createTheme({
        palette: {
            mode: isDark ? 'dark' : 'light',
            primary: { main: '#38bdf8' },
            background: { default: '#020617', paper: '#0f172a' }
        },
        typography: {
            fontFamily: '"Outfit", sans-serif'
        }
    });
};

export * from './components/GeminiLoader';
export * from './components/WindowLayout';

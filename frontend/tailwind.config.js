/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        // Design system tokens — used by name everywhere, never raw hex in components.
        ink: '#0A0A0B',      // primary background
        surface: '#131315',  // card/panel background
        border: {
          DEFAULT: '#1F1F22',
          strong: '#2A2A2E',
        },
        paper: '#F6F3EE',    // primary text on dark
        muted: {
          DEFAULT: '#9A9A9E',
          dim: '#6B6B70',
          faint: '#55555A',
        },
        accent: {
          DEFAULT: '#FF5A1F',
          hover: '#FF7A45',
        },
        success: '#3CCB7F',
        info: '#4C8DFF',
        warn: '#E8A33D',
      },
      fontFamily: {
        sans: ['"Space Grotesk"', 'system-ui', 'sans-serif'],
        mono: ['"IBM Plex Mono"', 'ui-monospace', 'monospace'],
        body: ['"Manrope"', 'system-ui', 'sans-serif'],
      },
      borderRadius: {
        DEFAULT: '10px',
        lg: '16px',
        xl: '20px',
      },
    },
  },
  plugins: [],
}

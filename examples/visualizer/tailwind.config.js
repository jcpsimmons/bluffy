/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./src/**/*.{js,jsx,ts,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        dark: {
          bg: '#0f0f23',
          surface: '#1a1a2e',
          card: '#16213e',
          hover: '#0e3460',
          border: '#2a2a54',
          text: '#e2e8f0',
          muted: '#94a3b8',
          accent: '#64748b',
          primary: '#3b82f6',
          secondary: '#8b5cf6',
        }
      },
      fontFamily: {
        mono: ['JetBrains Mono', 'Menlo', 'Monaco', 'Consolas', 'monospace'],
      },
      backdropBlur: {
        xs: '2px',
      }
    },
  },
  plugins: [],
}
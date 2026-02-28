/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        primary: '#0f1117',
        secondary: '#1a1d27',
        tertiary: '#242833',
        accent: '#6366f1',
        'accent-hover': '#818cf8',
        border: '#2e3345',
      },
    },
  },
  plugins: [],
};

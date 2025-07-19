import type { Config } from 'tailwindcss';

export default {
 content: ['./src/**/*.{ts,tsx}', './index.html'], // Ensure it scans App.tsx and others
 theme: {
 extend: {
 colors: {
 'navy-dark': '#0A0F1E',
 'navy-light': '#1A2338',
 'neon-blue': '#00FFFF',
 'neon-purple': '#9B5DE5',
 'neon-pink': '#FF007F',
 'club-gray': '#2A2F45',
 },
 fontFamily: {
 sans: ['Montserrat', 'sans-serif'],
 display: ['Montserrat', 'sans-serif'],
 },
 boxShadow: {
 'neon': '0 0 10px rgba(0, 255, 255, 0.5)',
 'neon-hover': '0 0 20px rgba(0, 255, 255, 0.8)',
 },
 backgroundImage: {
 'club-gradient': 'linear-gradient(to bottom, #0A0F1E, #1A2338)',
 },
 },
 },
 plugins: [],
} satisfies Config;
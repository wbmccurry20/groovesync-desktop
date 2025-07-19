// Updated main.tsx
// Changes: No major changes, but added import for global CSS if needed. Ensured React.StrictMode for dev.

import React from 'react'
import {createRoot} from 'react-dom/client'
import './style.css'
import './App.css' // Ensure both CSS files are imported
import App from './App'

const container = document.getElementById('root')

const root = createRoot(container!)

root.render(
    <React.StrictMode>
        <App/>
    </React.StrictMode>
)
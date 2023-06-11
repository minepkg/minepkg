import React from 'react'
import {createRoot} from 'react-dom/client'
import './style.css'
import App from './App'
import { WindowSetTitle } from '../wailsjs/runtime/runtime'

const container = document.getElementById('root')

const root = createRoot(container!)

WindowSetTitle('Minepkg test app');

root.render(
    <React.StrictMode>
        <App/>
    </React.StrictMode>
)

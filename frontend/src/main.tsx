import React from 'react'
import ReactDOM from 'react-dom/client'
import { App } from './App'
import { initI18n } from './i18n/i18n'
import { initLook } from './theme/useLook'
import './index.css'

initI18n()
initLook()

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)

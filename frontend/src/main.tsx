import React from 'react'
import ReactDOM from 'react-dom/client'
import { App } from './App'
import { initI18n } from './i18n/i18n'
import { defaultDesign } from './theme/designPreference'
import { initDesign } from './theme/useDesign'
import './index.css'

initI18n()
initDesign(defaultDesign(import.meta.env.VITE_DESIGN_DEFAULT))

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)

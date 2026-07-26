import { createContext, useContext, useState, useCallback, type ReactNode } from 'react'
import type { NavItem, ThemeMode } from '../types'

interface AppContextValue {
  activeNav: NavItem
  setActiveNav: (nav: NavItem) => void
  theme: ThemeMode
  toggleTheme: () => void
  commandOpen: boolean
  setCommandOpen: (open: boolean) => void
}

const AppContext = createContext<AppContextValue | null>(null)

export function AppProvider({ children }: { children: ReactNode }) {
  const [activeNav, setActiveNav] = useState<NavItem>('dashboard')
  const [theme, setTheme] = useState<ThemeMode>('dark')
  const [commandOpen, setCommandOpen] = useState(false)

  const toggleTheme = useCallback(() => {
    setTheme(prev => {
      const next = prev === 'dark' ? 'light' : 'dark'
      if (next === 'dark') {
        document.documentElement.classList.add('dark')
      } else {
        document.documentElement.classList.remove('dark')
      }
      return next
    })
  }, [])

  return (
    <AppContext.Provider value={{ activeNav, setActiveNav, theme, toggleTheme, commandOpen, setCommandOpen }}>
      {children}
    </AppContext.Provider>
  )
}

export function useApp() {
  const ctx = useContext(AppContext)
  if (!ctx) throw new Error('useApp outside AppProvider')
  return ctx
}

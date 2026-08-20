import { createContext, useContext, useState, useCallback, type ReactNode } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import type { NavItem, ThemeMode } from '../types'
import { navItemToPath, pathToNavItem } from '../routes'

interface AppContextValue {
  activeNav: NavItem
  setActiveNav: (nav: NavItem) => void
  // The URL is the source of truth for "current page" (so it survives a
  // refresh) -- this is the current location's history key, which changes
  // on every navigation, even re-navigating to the same path. Pages that
  // hold their own "detail view open" state (e.g. Products' editingId)
  // watch this to reset back to their list view when their nav item is
  // clicked again, since the path not changing would otherwise mean no
  // re-render to key off of.
  navSignal: string
  theme: ThemeMode
  toggleTheme: () => void
  commandOpen: boolean
  setCommandOpen: (open: boolean) => void
}

const AppContext = createContext<AppContextValue | null>(null)

export function AppProvider({ children }: { children: ReactNode }) {
  const location = useLocation()
  const navigate = useNavigate()
  const activeNav = pathToNavItem(location.pathname)
  const [theme, setTheme] = useState<ThemeMode>('dark')
  const [commandOpen, setCommandOpen] = useState(false)

  const setActiveNav = useCallback((nav: NavItem) => {
    navigate(navItemToPath(nav))
  }, [navigate])

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
    <AppContext.Provider value={{ activeNav, setActiveNav, navSignal: location.key, theme, toggleTheme, commandOpen, setCommandOpen }}>
      {children}
    </AppContext.Provider>
  )
}

export function useApp() {
  const ctx = useContext(AppContext)
  if (!ctx) throw new Error('useApp outside AppProvider')
  return ctx
}

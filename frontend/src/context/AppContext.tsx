import { createContext, useContext, useState, useCallback, useEffect, type ReactNode } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import type { NavItem } from '../types'
import { useTheme } from '../theme/useTheme'
import type { ResolvedTheme, ThemePreference } from '../theme/themePreference'
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
  theme: ResolvedTheme
  themePreference: ThemePreference
  cycleThemePreference: () => void
  commandOpen: boolean
  setCommandOpen: (open: boolean) => void
  // The sidebar as a drawer on small screens (always shown from lg up).
  navOpen: boolean
  setNavOpen: (open: boolean) => void
}

const AppContext = createContext<AppContextValue | null>(null)

export function AppProvider({ children }: { children: ReactNode }) {
  const location = useLocation()
  const navigate = useNavigate()
  const activeNav = pathToNavItem(location.pathname)
  const { theme, preference: themePreference, cyclePreference: cycleThemePreference } = useTheme()
  const [commandOpen, setCommandOpen] = useState(false)
  const [navOpen, setNavOpen] = useState(false)

  // Any navigation (a link, Back, the palette) closes the drawer.
  useEffect(() => { setNavOpen(false) }, [location.key])

  const setActiveNav = useCallback((nav: NavItem) => {
    navigate(navItemToPath(nav))
  }, [navigate])

  return (
    <AppContext.Provider value={{ activeNav, setActiveNav, navSignal: location.key, theme, themePreference, cycleThemePreference, commandOpen, setCommandOpen, navOpen, setNavOpen }}>
      {children}
    </AppContext.Provider>
  )
}

export function useApp() {
  const ctx = useContext(AppContext)
  if (!ctx) throw new Error('useApp outside AppProvider')
  return ctx
}

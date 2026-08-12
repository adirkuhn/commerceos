import { createContext, useContext, useState, useCallback, type ReactNode } from 'react'
import type { NavItem, ThemeMode } from '../types'

interface AppContextValue {
  activeNav: NavItem
  setActiveNav: (nav: NavItem) => void
  // Bumped on every setActiveNav call, even re-selecting the already-active
  // item. activeNav itself won't change in that case (same value bails out
  // of React's re-render), so pages that hold their own "detail view open"
  // state (e.g. Products' editingId) watch this to reset back to their list
  // view when their nav item is clicked again.
  navSignal: number
  theme: ThemeMode
  toggleTheme: () => void
  commandOpen: boolean
  setCommandOpen: (open: boolean) => void
}

const AppContext = createContext<AppContextValue | null>(null)

export function AppProvider({ children }: { children: ReactNode }) {
  const [activeNav, setActiveNavState] = useState<NavItem>('dashboard')
  const [navSignal, setNavSignal] = useState(0)
  const [theme, setTheme] = useState<ThemeMode>('dark')
  const [commandOpen, setCommandOpen] = useState(false)

  const setActiveNav = useCallback((nav: NavItem) => {
    setActiveNavState(nav)
    setNavSignal(n => n + 1)
  }, [])

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
    <AppContext.Provider value={{ activeNav, setActiveNav, navSignal, theme, toggleTheme, commandOpen, setCommandOpen }}>
      {children}
    </AppContext.Provider>
  )
}

export function useApp() {
  const ctx = useContext(AppContext)
  if (!ctx) throw new Error('useApp outside AppProvider')
  return ctx
}

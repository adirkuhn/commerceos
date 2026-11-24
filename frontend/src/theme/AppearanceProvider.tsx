import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import i18next from 'i18next'
import { apiErrorText } from '../api/apiErrorText'
import { getAppearance, updateAppearance } from '../settings/appearance/api'
import type { Appearance, AppearanceSettings } from './appearance'
import { applyAppearance, renderAppearance, saveAppliedAppearance } from './applyAppearance'
import { setShopDefaultLook } from './useLook'

export interface AppearanceState {
  // The shop's saved appearance; null until loaded.
  appearance: Appearance | null
  error: string
  // Shows a not-yet-saved look on the whole app (Settings' live preview);
  // null goes back to the saved one.
  preview: (look: AppearanceSettings | null) => void
  save: (look: AppearanceSettings) => Promise<Appearance>
}

const AppearanceContext = createContext<AppearanceState | null>(null)

function storage(): Storage | null {
  try {
    return window.localStorage
  } catch {
    return null
  }
}

function show(look: AppearanceSettings, remember: boolean) {
  const applied = renderAppearance(look)
  applyAppearance(document, applied)
  setShopDefaultLook(applied.look)
  if (remember) saveAppliedAppearance(storage(), applied)
}

// AppearanceProvider loads the shop's admin appearance and keeps it applied
// to the page (and cached for the next visit's first paint).
export function AppearanceProvider({ token, children }: { token: string; children: ReactNode }) {
  const [appearance, setAppearance] = useState<Appearance | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    getAppearance(token)
      .then(a => {
        if (cancelled) return
        setAppearance(a)
        show(a, true)
      })
      .catch(e => { if (!cancelled) setError(apiErrorText(e) ?? i18next.t('settings:appearance.loadFailed')) })
    return () => { cancelled = true }
  }, [token])

  const preview = useCallback((look: AppearanceSettings | null) => {
    const target = look ?? appearance
    if (target) show(target, false)
  }, [appearance])

  const save = useCallback(async (look: AppearanceSettings) => {
    const saved = await updateAppearance(token, look)
    setAppearance(saved)
    show(saved, true)
    return saved
  }, [token])

  const value = useMemo(() => ({ appearance, error, preview, save }), [appearance, error, preview, save])
  return <AppearanceContext.Provider value={value}>{children}</AppearanceContext.Provider>
}

export function useAppearance(): AppearanceState {
  const ctx = useContext(AppearanceContext)
  if (!ctx) throw new Error('useAppearance needs an AppearanceProvider')
  return ctx
}

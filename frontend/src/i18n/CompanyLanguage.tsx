import { useEffect } from 'react'
import * as settingsApi from '../settings/api'
import { applyCompanyLanguage } from './i18n'

/**
 * Switches the admin to the company language (Settings → Localization)
 * once it has loaded, for a user who never picked their own.
 */
export function CompanyLanguage({ token }: { token: string }) {
  useEffect(() => {
    if (!token) return
    let cancelled = false
    settingsApi.getTenantSettings(token)
      .then(s => { if (!cancelled) return applyCompanyLanguage(s.language) })
      .catch((e: unknown) => console.warn('company language unavailable, keeping the browser language', e))
    return () => { cancelled = true }
  }, [token])
  return null
}

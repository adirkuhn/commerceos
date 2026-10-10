import { useCallback, useEffect, useState } from 'react'
import i18next from 'i18next'
import { getCompanyProfile, updateCompanyProfile } from './api'
import type { CompanyProfile, CompanyProfileInput } from './types'
import { apiErrorText } from '../../api/apiErrorText'

export interface CompanyProfileState {
  profile: CompanyProfile | null
  loading: boolean
  error: string
  // Resolves with the saved profile; rejects with CompanyProfileError so
  // the caller can show it next to the offending field.
  save: (input: CompanyProfileInput) => Promise<CompanyProfile>
}

export function useCompanyProfile(token: string): CompanyProfileState {
  const [profile, setProfile] = useState<CompanyProfile | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    getCompanyProfile(token)
      .then(p => { if (!cancelled) setProfile(p) })
      .catch(e => { if (!cancelled) setError(apiErrorText(e) ?? i18next.t('settings:company.loadFailed')) })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token])

  const save = useCallback(async (input: CompanyProfileInput) => {
    const saved = await updateCompanyProfile(token, input)
    setProfile(saved)
    return saved
  }, [token])

  return { profile, loading, error, save }
}

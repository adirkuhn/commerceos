import i18next from 'i18next'
import { useCallback, useState } from 'react'
import { apiErrorText } from '../../api/apiErrorText'
import type { UpdateMoneybirdSettingsInput } from '../types'

export type SaveSettings = (input: UpdateMoneybirdSettingsInput) => Promise<void>

// One settings section's save: in flight, saved, or why it failed.
export function useSectionSave(save: SaveSettings) {
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const run = useCallback(async (input: UpdateMoneybirdSettingsInput) => {
    setSaving(true)
    setSaved(false)
    setError(null)
    try {
      await save(input)
      setSaved(true)
    } catch (err) {
      setError(apiErrorText(err) ?? i18next.t('integrations:moneybird.saveFailed'))
    } finally {
      setSaving(false)
    }
  }, [save])

  return { saving, saved, error, run }
}

import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../../auth/AuthContext'
import { apiErrorText } from '../../api/apiErrorText'
import * as api from '../api'
import type { MessageTemplateDefault } from '../types'

interface Props {
  value: string
  onChoose: (d: MessageTemplateDefault | null) => void
}

// StartFromSelect lets a new template start from a built-in default (in
// the store's language) or blank, and says in one line what the chosen
// default does.
export function StartFromSelect({ value, onChoose }: Props) {
  const { token } = useAuth()
  const { t } = useTranslation(['messages', 'common'])
  const [defaults, setDefaults] = useState<MessageTemplateDefault[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    api.listMessageTemplateDefaults(token ?? '').then(setDefaults, err => setError(apiErrorText(err) ?? t('common:status.error')))
  }, [token, t])

  const chosen = defaults.find(d => d.key === value)
  return (
    <div>
      <label htmlFor="template-start-from" className="block text-2xs text-ink-4 caps-label mb-1.5">
        {t('templates.startFrom')}
      </label>
      <select
        id="template-start-from"
        value={value}
        onChange={e => onChoose(defaults.find(d => d.key === e.target.value) ?? null)}
        className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
      >
        <option value="">{t('templates.startBlank')}</option>
        {defaults.map(d => <option key={d.key} value={d.key}>{d.name}</option>)}
      </select>
      <p className="text-2xs text-ink-4 mt-1">{chosen ? chosen.description : t('templates.startBlankHint')}</p>
      {error && <p role="alert" className="text-2xs text-danger mt-1">{error}</p>}
    </div>
  )
}

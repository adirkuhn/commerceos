import { useEffect, useRef, useState, type FormEvent } from 'react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import { useAuth } from '../../auth/AuthContext'
import { NotificationPreferencesError } from './api'
import { useNotificationPreferences } from './useNotificationPreferences'
import type { NotificationCategory, NotificationCategoryPreference, NotificationPreferencesInput } from './types'
import { useDateFormat } from '../../format/DateFormatProvider'
import { apiErrorText } from '../../api/apiErrorText'

function categoryText(t: TFunction<['settings', 'common']>, category: NotificationCategory): { name: string; description: string } {
  switch (category) {
    case 'imports': return { name: t('notifications.imports'), description: t('notifications.importsText') }
    case 'purchase_documents': return { name: t('notifications.purchaseDocuments'), description: t('notifications.purchaseDocumentsText') }
    case 'channel_offers': return { name: t('notifications.channelOffers'), description: t('notifications.channelOffersText') }
    case 'stock': return { name: t('notifications.stock'), description: t('notifications.stockText') }
    case 'other': return { name: t('notifications.other'), description: t('notifications.otherText') }
  }
}

// Matches the backend's RequireRoleForWrites on the settings routes.
const EDITOR_ROLES = ['customer_owner', 'customer_admin', 'super_admin']

function parseRecipients(text: string): string[] {
  return text.split(/[\n,;]/).map(s => s.trim()).filter(Boolean)
}

// Settings > Notifications: per category, whether it shows in the bell
// and/or is emailed, and to whom.
export function NotificationPreferencesSection() {
  const { token, user } = useAuth()
  const { t } = useTranslation(['settings', 'common'])
  const { formatTime } = useDateFormat()
  const { preferences, loading, error: loadError, save } = useNotificationPreferences(token ?? '')
  const canEdit = EDITOR_ROLES.includes(user?.role ?? '')

  // Edits only (null = show what's saved) -- derived, not copied in an
  // effect, so the first render after loading already shows saved values.
  const [editedCategories, setEditedCategories] = useState<NotificationCategoryPreference[] | null>(null)
  const [editedRecipients, setEditedRecipients] = useState<string | null>(null)
  const categories = editedCategories ?? preferences?.categories ?? []
  const recipientsText = editedRecipients ?? preferences?.email_recipients.join('\n') ?? ''
  const [saving, setSaving] = useState(false)
  const [recipientError, setRecipientError] = useState('')
  const [formError, setFormError] = useState('')
  const [savedAt, setSavedAt] = useState<string | null>(null)
  const recipientsRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => { if (recipientError) recipientsRef.current?.focus() }, [recipientError])

  function toggle(category: NotificationCategory, channel: 'in_app' | 'email') {
    setEditedCategories(categories.map(c => (c.category === category ? { ...c, [channel]: !c[channel] } : c)))
    setSavedAt(null)
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    const input: NotificationPreferencesInput = { categories, email_recipients: parseRecipients(recipientsText) }
    setSaving(true)
    setRecipientError('')
    setFormError('')
    try {
      await save(input)
      setEditedCategories(null)
      setEditedRecipients(null)
      setSavedAt(formatTime(new Date()))
    } catch (err) {
      if (err instanceof NotificationPreferencesError && err.isRecipientError) setRecipientError(err.message)
      else setFormError(apiErrorText(err) ?? t('notifications.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <p className="text-sm text-ink-4">{t('notifications.loading')}</p>
  if (loadError) return <p role="alert" className="text-sm text-danger">{loadError}</p>

  return (
    <form onSubmit={handleSubmit} className="max-w-2xl space-y-6" noValidate>
      <div>
        <h2 className="text-base font-semibold text-ink mb-1">{t('sections.notifications')}</h2>
        <p className="text-xs text-ink-4">{t('notifications.subtitle')}</p>
        {!canEdit && <p className="mt-2 text-xs text-ink-3">{t('notifications.readOnly')}</p>}
      </div>

      {formError && <p role="alert" className="text-sm text-danger">{formError}</p>}

      <table className="w-full text-sm bg-raised border border-line rounded-xl overflow-hidden">
        <thead>
          <tr className="border-b border-line text-xs text-ink-3">
            <th scope="col" className="text-left font-medium px-4 py-2.5">{t('notifications.category')}</th>
            <th scope="col" className="w-20 font-medium px-2 py-2.5">{t('notifications.bell')}</th>
            <th scope="col" className="w-20 font-medium px-2 py-2.5">{t('common:fields.email')}</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-line">
          {categories.map(c => {
            const text = categoryText(t, c.category)
            const lower = text.name.toLowerCase()
            return (
              <tr key={c.category}>
                <th scope="row" className="text-left font-normal px-4 py-3">
                  <span className="block text-ink">{text.name}</span>
                  <span className="block text-xs text-ink-4">{text.description}</span>
                </th>
                <td className="text-center px-2">
                  <input type="checkbox" aria-label={t('notifications.showInBell', { category: lower })} checked={c.in_app}
                    disabled={!canEdit} onChange={() => toggle(c.category, 'in_app')} className="w-4 h-4 accent-accent-solid" />
                </td>
                <td className="text-center px-2">
                  <input type="checkbox" aria-label={t('notifications.emailCategory', { category: lower })} checked={c.email}
                    disabled={!canEdit} onChange={() => toggle(c.category, 'email')} className="w-4 h-4 accent-accent-solid" />
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>

      <div>
        <label htmlFor="notification-recipients" className="block text-xs font-medium text-ink-2 mb-1.5">{t('notifications.recipients')}</label>
        <textarea
          id="notification-recipients"
          ref={recipientsRef}
          value={recipientsText}
          onChange={e => { setEditedRecipients(e.target.value); setSavedAt(null) }}
          readOnly={!canEdit}
          rows={3}
          placeholder={'ops@example.com\nowner@example.com'}
          aria-invalid={recipientError ? true : undefined}
          aria-describedby={clsx('notification-recipients-hint', recipientError && 'notification-recipients-error')}
          className={clsx(
            'w-full px-3 py-2 text-sm rounded-lg bg-surface border text-ink placeholder-ink-4 focus:outline-none focus:ring-2 focus:ring-focus/60 read-only:bg-sunken',
            recipientError ? 'border-danger' : 'border-line',
          )}
        />
        <p id="notification-recipients-hint" className="mt-1 text-xs text-ink-4">{t('notifications.recipientsHint')}</p>
        {recipientError && <p id="notification-recipients-error" className="mt-1 text-xs text-danger">{recipientError}</p>}
      </div>

      {canEdit && (
        <div className="flex items-center gap-3">
          <button type="submit" disabled={saving}
            className="px-4 py-2 rounded-lg bg-accent-solid text-white text-sm font-medium hover:bg-accent-solid-hover disabled:opacity-50">
            {saving ? t('common:actions.saving') : t('common:actions.saveChanges')}
          </button>
          <p role="status" className="text-xs text-ink-3">{savedAt && t('savedAt', { time: savedAt })}</p>
        </div>
      )}
    </form>
  )
}

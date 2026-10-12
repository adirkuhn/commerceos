import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { LOCALES, LOCALE_NAMES, resolveLocale, type Locale } from '../i18n/locale'
import { useTenantSettings } from '../settings/hooks/useTenantSettings'
import type { CollectionTexts } from './types'

const inputClass = 'w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus'

/** The collection's texts in the languages other than the company one,
 *  saved with the collection. Blank shows the default text (the hints). */
export function TranslationsFieldset({ value, fallback, onChange }: {
  value: Record<string, CollectionTexts>
  fallback: Required<CollectionTexts>
  onChange: (next: Record<string, CollectionTexts>) => void
}) {
  const { token } = useAuth()
  const { t } = useTranslation('collections')
  const { settings } = useTenantSettings(token ?? '')
  const [active, setActive] = useState<Locale>()
  if (!settings) return null
  const languages = LOCALES.filter(l => l !== resolveLocale(settings.language))
  const current = active && languages.includes(active) ? active : languages[0]
  const texts = value[current] ?? {}
  const set = (field: keyof CollectionTexts, v: string) => onChange({ ...value, [current]: { ...texts, [field]: v } })

  return (
    <fieldset className="grid gap-3">
      <legend className="block text-2xs text-ink-4 caps-label mb-1.5">{t('edit.translations')}</legend>
      <div role="tablist" className="flex gap-1 border-b border-line">
        {languages.map(l => (
          <button key={l} type="button" role="tab" aria-selected={l === current} onClick={() => setActive(l)}
            className={clsx('px-3 py-1.5 text-xs', l === current ? 'border-b-2 border-accent-solid font-medium text-ink' : 'text-ink-4')}>
            {LOCALE_NAMES[l]}
          </button>
        ))}
      </div>
      <input aria-label={t('edit.inLanguage', { field: t('edit.name'), language: LOCALE_NAMES[current] })} className={inputClass} maxLength={120} value={texts.name ?? ''} placeholder={fallback.name} onChange={e => set('name', e.target.value)} />
      <textarea aria-label={t('edit.inLanguage', { field: t('edit.description'), language: LOCALE_NAMES[current] })} className={inputClass} maxLength={5000} value={texts.description ?? ''} placeholder={fallback.description} onChange={e => set('description', e.target.value)} />
      <input aria-label={t('edit.inLanguage', { field: t('edit.seoTitle'), language: LOCALE_NAMES[current] })} className={inputClass} maxLength={70} value={texts.seo_title ?? ''} placeholder={fallback.seo_title} onChange={e => set('seo_title', e.target.value)} />
      <textarea aria-label={t('edit.inLanguage', { field: t('edit.seoDescription'), language: LOCALE_NAMES[current] })} className={inputClass} maxLength={320} value={texts.seo_description ?? ''} placeholder={fallback.seo_description} onChange={e => set('seo_description', e.target.value)} />
    </fieldset>
  )
}

import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { LOCALES, LOCALE_NAMES, resolveLocale, type Locale } from '../i18n/locale'
import { useTenantSettings } from '../settings/hooks/useTenantSettings'
import { setProductTranslations } from './api'
import { FormCard } from './FormCard'
import { inputClass, labelClass } from './formStyles'
import type { Product, ProductTexts } from './types'

type Translations = Partial<Record<Locale, ProductTexts>>

/** The product's name and description in the languages other than the
 *  company one (shoppers switching the store's language see them). Saved on
 *  its own; a blank text shows the product's own. */
export function ProductTranslationsCard({ product }: { product: Product }) {
  const { token } = useAuth()
  const { t } = useTranslation(['pms', 'common'])
  const { settings } = useTenantSettings(token ?? '')
  const company = resolveLocale(settings?.language)
  const languages = LOCALES.filter(l => l !== company)
  const [active, setActive] = useState<Locale>()
  const [values, setValues] = useState<Translations>(product.translations ?? {})
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)
  const current = active && languages.includes(active) ? active : languages[0]

  useEffect(() => { setValues(product.translations ?? {}) }, [product.translations])

  const set = (field: keyof ProductTexts, v: string) => {
    setSaved(false)
    setValues(prev => ({ ...prev, [current]: { ...prev[current], [field]: v } }))
  }

  async function save() {
    setSaving(true)
    setError('')
    try {
      const p = await setProductTranslations(token ?? '', product.id, values)
      setValues(p.translations ?? {})
      setSaved(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('translations.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  if (!settings) return null
  const texts = values[current] ?? {}
  return (
    <FormCard title={t('translations.title')} description={t('translations.help', { language: LOCALE_NAMES[company] })}>
      <div role="tablist" className="flex gap-1 border-b border-line">
        {languages.map(l => (
          <button key={l} type="button" role="tab" aria-selected={l === current} onClick={() => setActive(l)}
            className={clsx('px-3 py-1.5 text-xs', l === current ? 'border-b-2 border-accent-solid font-medium text-ink' : 'text-ink-4')}>
            {LOCALE_NAMES[l]}
          </button>
        ))}
      </div>
      <div>
        <label htmlFor="translation-name" className={labelClass}>{t('translations.name')}</label>
        <input id="translation-name" type="text" maxLength={255} value={texts.name ?? ''} placeholder={product.name}
          onChange={e => set('name', e.target.value)} className={inputClass} />
      </div>
      <div>
        <label htmlFor="translation-description" className={labelClass}>{t('translations.description')}</label>
        <textarea id="translation-description" rows={3} maxLength={20000} value={texts.description ?? ''} placeholder={product.description}
          onChange={e => set('description', e.target.value)} className={inputClass} />
      </div>
      {error && <p role="alert" className="text-xs text-danger">{error}</p>}
      {saved && <p className="text-xs text-success">{t('translations.saved')}</p>}
      <button type="button" onClick={() => void save()} disabled={saving}
        className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium disabled:opacity-50">
        {saving ? t('common:actions.saving') : t('translations.save')}
      </button>
    </FormCard>
  )
}

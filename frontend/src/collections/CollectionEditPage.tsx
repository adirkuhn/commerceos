import { useEffect, useState } from 'react'
import { TranslationsFieldset } from './TranslationsFieldset'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import { ArrowLeft } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import { getProduct } from '../pms/api'
import { emptyInput, slugify, toInput, validationMessage, withType } from './form'
import { ManualProducts } from './ManualProducts'
import { RulesEditor } from './RulesEditor'
import { inputClass, labelClass } from './styles'
import type { Collection, CollectionInput, CollectionSort } from './types'
import { apiErrorText } from '../api/apiErrorText'

const sorts = (t: TFunction<'collections'>): { value: CollectionSort; label: string; manualOnly?: boolean }[] => [
  { value: 'manual', label: t('sort.manual'), manualOnly: true },
  { value: 'newest', label: t('sort.newest') },
  { value: 'name_asc', label: t('sort.nameAsc') },
  { value: 'price_asc', label: t('sort.priceAsc') },
  { value: 'price_desc', label: t('sort.priceDesc') },
]

interface Props {
  collection: Collection | null
  onSave: (id: string | null, input: CollectionInput, productIds?: string[]) => Promise<Collection>
  onClose: () => void
  onSaved: (id: string) => void
}

export function CollectionEditPage({ collection, onSave, onClose, onSaved }: Props) {
  const { token } = useAuth()
  const { t } = useTranslation(['collections', 'common'])
  const [input, setInput] = useState<CollectionInput>(() => (collection ? toInput(collection) : emptyInput()))
  const [slugTouched, setSlugTouched] = useState(!!collection)
  const [productIds, setProductIds] = useState<string[]>(collection?.product_ids ?? [])
  const [names, setNames] = useState<Record<string, string>>({})
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const set = (patch: Partial<CollectionInput>) => setInput(i => ({ ...i, ...patch }))

  useEffect(() => {
    const missing = (collection?.product_ids ?? []).filter(id => !names[id])
    if (!missing.length || !token) return
    Promise.all(missing.map(id => getProduct(token, id).then(p => [id, p.product.name] as const, () => [id, id] as const)))
      .then(pairs => setNames(n => ({ ...n, ...Object.fromEntries(pairs) })))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [collection, token])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const problem = validationMessage(input)
    if (problem) {
      setError(problem)
      return
    }
    setSaving(true)
    setError('')
    try {
      const saved = await onSave(collection?.id ?? null, input, input.type === 'manual' ? productIds : undefined)
      onSaved(saved.id)
    } catch (err) {
      setError(apiErrorText(err) ?? t('edit.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col h-full overflow-y-auto">
      <div className="flex items-center gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <button type="button" onClick={onClose} aria-label={t('edit.back')} className="p-1.5 rounded-lg bg-fill text-ink-3 hover:text-accent"><ArrowLeft size={14} /></button>
        <h2 className="text-sm font-semibold text-ink">{collection ? collection.name : t('edit.new')}</h2>
        <label className="ml-auto flex items-center gap-2 text-xs text-ink-2">
          <input type="checkbox" role="switch" checked={!!input.published} onChange={e => set({ published: e.target.checked })} />
          {t('edit.visible')}
        </label>
        <button type="submit" disabled={saving} className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium disabled:opacity-50">
          {saving ? t('common:actions.saving') : t('common:actions.save')}
        </button>
      </div>
      {error && <p role="alert" className="mx-5 mt-3 text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-3 py-2">{error}</p>}
      <div className="grid gap-5 p-5 max-w-3xl">
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label htmlFor="col-name" className={labelClass}>{t('edit.name')}</label>
            <input id="col-name" className={inputClass} value={input.name} maxLength={120}
              onChange={e => set({ name: e.target.value, ...(slugTouched ? {} : { slug: slugify(e.target.value) }) })} />
          </div>
          <div>
            <label htmlFor="col-slug" className={labelClass}>{t('edit.url')}</label>
            <div className="flex items-center gap-1 text-xs text-ink-4">{'/collections/'}
              <input id="col-slug" className={inputClass} value={input.slug} maxLength={64}
                onChange={e => { setSlugTouched(true); set({ slug: e.target.value }) }} />
            </div>
          </div>
        </div>
        <div>
          <label htmlFor="col-description" className={labelClass}>{t('edit.description')}</label>
          <textarea id="col-description" className={`${inputClass} min-h-20`} value={input.description ?? ''} maxLength={5000}
            onChange={e => set({ description: e.target.value })} />
        </div>
        <fieldset className="space-y-3">
          <legend className={labelClass}>{t('edit.products')}</legend>
          <div className="flex gap-4 text-xs text-ink-2">
            <label className="flex items-center gap-1.5"><input type="radio" name="col-type" checked={input.type === 'manual'} onChange={() => setInput(i => withType(i, 'manual'))} /> {t('edit.manual')}</label>
            <label className="flex items-center gap-1.5"><input type="radio" name="col-type" checked={input.type === 'smart'} onChange={() => setInput(i => withType(i, 'smart'))} /> {t('edit.smart')}</label>
          </div>
          {input.type === 'manual'
            ? <ManualProducts ids={productIds} names={names} onChange={(ids, n) => { setProductIds(ids); setNames(n) }} />
            : <RulesEditor match={input.match ?? 'all'} rules={input.rules ?? []} onMatch={match => set({ match })} onRules={rules => set({ rules })} />}
        </fieldset>
        <div>
          <label htmlFor="col-sort" className={labelClass}>{t('edit.sort')}</label>
          <select id="col-sort" className={inputClass} value={input.sort} onChange={e => set({ sort: e.target.value as CollectionSort })}>
            {sorts(t).filter(s => input.type === 'manual' || !s.manualOnly).map(s => <option key={s.value} value={s.value}>{s.label}</option>)}
          </select>
        </div>
        <fieldset className="grid gap-3">
          <legend className={labelClass}>{t('edit.seo')}</legend>
          <input aria-label={t('edit.seoTitle')} placeholder={t('edit.seoTitlePlaceholder')} className={inputClass} value={input.seo_title ?? ''} maxLength={70} onChange={e => set({ seo_title: e.target.value })} />
          <textarea aria-label={t('edit.seoDescription')} placeholder={t('edit.seoDescriptionPlaceholder')} className={inputClass} value={input.seo_description ?? ''} maxLength={320} onChange={e => set({ seo_description: e.target.value })} />
        </fieldset>
        <TranslationsFieldset value={input.translations ?? {}} onChange={translations => set({ translations })}
          fallback={{ name: input.name, description: input.description ?? '', seo_title: input.seo_title ?? '', seo_description: input.seo_description ?? '' }} />
      </div>
    </form>
  )
}

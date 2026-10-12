import { useTranslation } from 'react-i18next'
import type { DocumentKind, DocumentTexts, SEO } from '../../core/document'

/** The open document's own texts: title (pages) and search-engine texts,
 *  plus their translations. */
export interface PageTexts {
  title: string
  seo: SEO
  translations?: Record<string, DocumentTexts>
}

const input = 'w-full rounded-ed border border-ed-line bg-ed-sunken px-2.5 py-1.5 text-sm text-ed-ink focus:outline-none focus:ring-2 focus:ring-ed-accent/60'

/** Edits the default-language texts, or -- with a locale -- that
 *  language's translation (blank shows the default, given as a hint). */
export function PageTextsForm({ kind, texts, locale, onChange }: { kind: DocumentKind; texts: PageTexts; locale: string; onChange: (t: PageTexts) => void }) {
  const { t } = useTranslation('editor')
  const current: DocumentTexts = locale ? texts.translations?.[locale] ?? {} : texts
  const set = (next: DocumentTexts) => {
    if (!locale) return onChange({ ...texts, title: next.title ?? '', seo: next.seo ?? {} })
    onChange({ ...texts, translations: { ...texts.translations, [locale]: { title: next.title, seo: next.seo ?? {} } } })
  }
  const hint = (v: string | undefined) => (locale ? v : undefined)
  return (
    <section className="flex flex-col gap-3 border-b pb-4">
      <h2 className="font-semibold">{t('page.heading')}</h2>
      {kind === 'page' && (
        <label className="flex flex-col gap-1 text-sm">{t('page.title')}
          <input className={input} maxLength={120} value={current.title ?? ''} placeholder={hint(texts.title)}
            onChange={e => set({ ...current, title: e.target.value })} />
        </label>
      )}
      <label className="flex flex-col gap-1 text-sm">{t('page.seoTitle')}
        <input className={input} maxLength={70} value={current.seo?.title ?? ''} placeholder={hint(texts.seo.title)}
          onChange={e => set({ ...current, seo: { ...current.seo, title: e.target.value } })} />
      </label>
      <label className="flex flex-col gap-1 text-sm">{t('page.seoDescription')}
        <textarea className={`${input} min-h-20`} maxLength={320} value={current.seo?.description ?? ''} placeholder={hint(texts.seo.description)}
          onChange={e => set({ ...current, seo: { ...current.seo, description: e.target.value } })} />
      </label>
      <p className="text-xs text-ed-ink-4">{t('page.seoHelp')}</p>
    </section>
  )
}

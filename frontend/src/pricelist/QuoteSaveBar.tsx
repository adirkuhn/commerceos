import { CopyPlus, FilePlus2, Save } from 'lucide-react'
import { useTranslation } from 'react-i18next'

export interface QuoteSaveBarProps {
  reference: string | null
  name: string
  onName: (name: string) => void
  dirty: boolean
  saving: boolean
  error: string
  onSave: () => void
  onSaveAsNew: () => void
  onNew: () => void
}

// Name, number and save actions of the quote in the editor.
export function QuoteSaveBar({ reference, name, onName, dirty, saving, error, onSave, onSaveAsNew, onNew }: QuoteSaveBarProps) {
  const { t } = useTranslation('priceList')
  const canSave = name.trim() !== '' && !saving && (dirty || reference === null)

  return (
    <div className="flex flex-wrap items-center gap-2 px-5 py-2 border-b border-line bg-surface shrink-0">
      {reference && <span data-testid="quote-reference" className="px-2 py-0.5 rounded-md bg-sunken font-mono text-2xs text-ink-2">{reference}</span>}
      <input
        value={name}
        onChange={e => onName(e.target.value)}
        onKeyDown={e => { if (e.key === 'Enter' && canSave) onSave() }}
        placeholder={t('quote.namePlaceholder')}
        aria-label={t('quote.name')}
        maxLength={200}
        className="w-64 py-1.5 px-2.5 text-xs rounded-lg bg-raised border border-line text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-focus"
      />
      {dirty && <span data-testid="quote-unsaved" className="text-2xs text-warning">● {t('quote.unsaved')}</span>}
      {error && <span role="alert" className="text-xs text-danger">{error}</span>}
      <div className="flex items-center gap-2 ml-auto">
        <button onClick={onNew} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-line text-xs text-ink-2 hover:bg-hover">
          <FilePlus2 size={13} /> {t('quote.newQuote')}
        </button>
        {reference && (
          <button onClick={onSaveAsNew} disabled={saving || name.trim() === ''}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-line text-xs text-ink-2 hover:bg-hover disabled:opacity-40">
            <CopyPlus size={13} /> {t('quote.saveAsNew')}
          </button>
        )}
        <button onClick={onSave} disabled={!canSave} title={name.trim() === '' ? t('quote.nameNeeded') : undefined}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-40">
          <Save size={13} /> {saving ? t('quote.saving') : t('quote.save')}
        </button>
      </div>
    </div>
  )
}

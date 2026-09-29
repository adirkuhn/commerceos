import { Trans, useTranslation } from 'react-i18next'
import { useState } from 'react'
import { Search } from 'lucide-react'
import { FormCard } from './FormCard'
import { CaseOfSKUPicker } from './VariantsSection'
import { inputClass, labelClass } from './formStyles'
import type { FlatSkuForm, PatchFlatSku, SetFlatSkuField } from './flatSkuForm'

interface FlatSkuCasePackCardProps {
  form: Pick<FlatSkuForm, 'caseOfSKUID' | 'caseOfSKULabel' | 'unitsPerCase'>
  setField: SetFlatSkuField
  patch: PatchFlatSku
  // The SKU being edited -- it can't be a case of itself.
  currentSkuId: string
}

// "Case pack" card: links this SKU as a sealed case of another SKU.
export function FlatSkuCasePackCard({ form, setField, patch, currentSkuId }: FlatSkuCasePackCardProps) {
  const [casePickerOpen, setCasePickerOpen] = useState(false)
  const { t } = useTranslation('pms')
  return (
    <FormCard title={t('casePack.title')} description={t('casePack.description')}>
      {form.caseOfSKUID ? (
        <div className="flex items-center gap-2">
          <span className="flex-1 text-xs text-ink">
            <Trans t={t} i18nKey="casePack.caseOf" values={{ label: form.caseOfSKULabel || form.caseOfSKUID }} components={{ b: <span className="font-medium" /> }} />
          </span>
          <div className="w-24">
            <label htmlFor="product-units-per-case" className={labelClass}>{t('casePack.unitsPerCase')}</label>
            <input
              id="product-units-per-case"
              type="number" step="1" min="1" required value={form.unitsPerCase} onChange={e => setField('unitsPerCase', e.target.value)}
              className={inputClass}
            />
          </div>
          <button
            type="button"
            onClick={() => patch({ caseOfSKUID: null, caseOfSKULabel: '', unitsPerCase: '' })}
            className="text-2xs text-ink-4 hover:underline"
          >
            {t('casePack.remove')}
          </button>
        </div>
      ) : casePickerOpen ? (
        <CaseOfSKUPicker
          excludeSKUID={currentSkuId}
          onSelect={(sku, label) => { patch({ caseOfSKUID: sku.id, caseOfSKULabel: label }); setCasePickerOpen(false) }}
          onClose={() => setCasePickerOpen(false)}
        />
      ) : (
        <button
          type="button"
          onClick={() => setCasePickerOpen(true)}
          className="flex items-center gap-1 text-xs font-medium text-accent hover:text-accent/80 transition-colors"
        >
          <Search size={12} /> {t('casePack.link')}
        </button>
      )}
    </FormCard>
  )
}

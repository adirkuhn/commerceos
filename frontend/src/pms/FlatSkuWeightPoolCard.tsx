import { Trans, useTranslation } from 'react-i18next'
import { useState } from 'react'
import { Search } from 'lucide-react'
import { FormCard } from './FormCard'
import { CaseOfSKUPicker } from './VariantsSection'
import { inputClass, labelClass } from './formStyles'
import type { FlatSkuForm, PatchFlatSku } from './flatSkuForm'
import type { SKUUnitType } from './types'

interface FlatSkuWeightPoolCardProps {
  form: Pick<FlatSkuForm, 'unitType' | 'portionedFromSKUID' | 'portionedFromLabel'>
  patch: PatchFlatSku
  // The SKU being edited (if saved yet) -- it can't be portioned from itself.
  currentSkuId: string | null
}

// "Weight pool" card: a SKU is either a weight pool itself (unit_type
// kilo), or portioned from one, never both.
export function FlatSkuWeightPoolCard({ form, patch, currentSkuId }: FlatSkuWeightPoolCardProps) {
  const [portionPickerOpen, setPortionPickerOpen] = useState(false)
  const { t } = useTranslation('pms')
  const { unitType, portionedFromSKUID, portionedFromLabel } = form
  return (
    <FormCard title={t('pool.title')} description={t('pool.description')}>
      <div>
        <label htmlFor="product-unit-type" className={labelClass}>{t('pool.itemType')}</label>
        <select
          id="product-unit-type" value={unitType}
          onChange={e => {
            const next = e.target.value as SKUUnitType
            patch(next === 'kilo' ? { unitType: next, portionedFromSKUID: null, portionedFromLabel: '' } : { unitType: next })
          }}
          className={inputClass}
        >
          <option value="unit">{t('pool.unit')}</option>
          <option value="box">{t('pool.box')}</option>
          <option value="kilo">{t('pool.kilo')}</option>
        </select>
      </div>
      {unitType !== 'kilo' && (
        portionedFromSKUID ? (
          <div className="flex items-center gap-2 mt-2">
            <span className="flex-1 text-xs text-ink">
              <Trans t={t} i18nKey="pool.portionedFrom" values={{ label: portionedFromLabel || portionedFromSKUID }} components={{ b: <span className="font-medium" /> }} />
            </span>
            <button
              type="button"
              onClick={() => patch({ portionedFromSKUID: null, portionedFromLabel: '' })}
              className="text-2xs text-ink-4 hover:underline"
            >
              {t('pool.remove')}
            </button>
          </div>
        ) : portionPickerOpen ? (
          <div className="mt-2">
            <CaseOfSKUPicker
              excludeSKUID={currentSkuId ?? ''}
              filterSku={s => s.unit_type === 'kilo'}
              onSelect={(sku, label) => { patch({ portionedFromSKUID: sku.id, portionedFromLabel: label }); setPortionPickerOpen(false) }}
              onClose={() => setPortionPickerOpen(false)}
            />
          </div>
        ) : (
          <button
            type="button"
            onClick={() => setPortionPickerOpen(true)}
            className="flex items-center gap-1 text-xs font-medium text-accent hover:text-accent/80 transition-colors mt-2"
          >
            <Search size={12} /> {t('pool.linkPool')}
          </button>
        )
      )}
    </FormCard>
  )
}

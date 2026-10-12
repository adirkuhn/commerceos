import { useState, type FormEvent } from 'react'
import { X } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { Dialog } from '../../components/common/Dialog'
import { useDateFormat } from '../../format/DateFormatProvider'
import { formatMoney } from '../../format/money'
import { addDays, zonedMidnight } from '../../format/calendarDay'
import { apiErrorText } from '../../api/apiErrorText'
import { createPromotion } from '../../promotions/api'
import { buildClearancePromotion, clearanceLine, PRICE_MODES, previewSalePrice, type PriceMode } from './clearance'
import type { CloseToExpiryBatch } from './types'

export interface ClearanceCreated {
  promotionId: string
  name: string
  warnings: string[]
}

const MODE_LABEL = { price: 'modePrice', percent: 'modePercent', amount: 'modeAmount' } as const
const VALUE_LABEL = { price: 'valuePrice', percent: 'valuePercent', amount: 'valueAmount' } as const
const INVALID = { price: 'invalidPrice', percent: 'invalidPercent', amount: 'invalidAmount' } as const

const inputClass = 'w-full px-3 py-2 text-sm rounded-lg bg-sunken border border-line text-ink outline-none focus:ring-1 focus:ring-focus'
const labelClass = 'block text-xs text-ink-3 mb-1.5'

// "Put on sale" for one close-to-expiry batch: an active clearance
// promotion for its SKU, starting now, that ends by itself once the batch
// is finished (see promotions' clearance lines). Amounts here stay
// unmasked: it is an edit form.
export function ClearancePromotionDialog({ token, batch, onClose, onCreated }: {
  token: string
  batch: CloseToExpiryBatch
  onClose: () => void
  onCreated: (created: ClearanceCreated) => void
}) {
  const { t } = useTranslation(['inventory', 'common'])
  const { formatDate, settings } = useDateFormat()
  const [mode, setMode] = useState<PriceMode>('price')
  const [value, setValue] = useState('')
  const [endsOn, setEndsOn] = useState(batch.expiry_date)
  const [name, setName] = useState(() => batch.lot_number
    ? t('closeToExpiry.dialog.defaultName', { product: batch.product_name, lot: batch.lot_number })
    : t('closeToExpiry.dialog.defaultNameNoLot', { product: batch.product_name }))
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const line = clearanceLine(mode, value)
  const money = (minor: number) => formatMoney(minor, batch.currency_code)

  async function submit(e: FormEvent) {
    e.preventDefault()
    if (!line) return setError(t(`closeToExpiry.dialog.${INVALID[mode]}`))
    if (!name.trim()) return setError(t('closeToExpiry.dialog.nameRequired'))
    setError('')
    setSubmitting(true)
    try {
      // The sale runs through the whole end day, in the shop's time zone.
      const endsAt = endsOn ? zonedMidnight(addDays(endsOn, 1), settings.timeZone) : null
      const res = await createPromotion(token, buildClearancePromotion({ batch, line, name, endsAt, now: new Date() }))
      onCreated({ promotionId: res.promotion.id, name: name.trim(), warnings: res.warnings })
    } catch (err) {
      setError(apiErrorText(err) ?? t('errors.unknown'))
      setSubmitting(false)
    }
  }

  const context = { lot: batch.lot_number, count: batch.quantity, date: formatDate(batch.expiry_date) }

  return (
    <Dialog onClose={onClose} dismissible={!submitting} label={t('closeToExpiry.dialog.title', { product: batch.product_name })}
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4">
      <form onSubmit={submit} noValidate className="w-full max-w-md rounded-2xl bg-raised border border-line p-5 space-y-4">
        <div className="flex items-start justify-between gap-3">
          <div>
            <h2 className="text-sm font-semibold text-ink">{t('closeToExpiry.dialog.title', { product: batch.product_name })}</h2>
            <p className="text-xs text-ink-4 mt-1 leading-relaxed">
              {batch.lot_number ? t('closeToExpiry.dialog.context', context) : t('closeToExpiry.dialog.contextNoLot', context)}
            </p>
          </div>
          <button type="button" aria-label={t('common:actions.close')} onClick={onClose} disabled={submitting}
            className="shrink-0 text-ink-4 hover:text-ink-2">
            <X size={16} />
          </button>
        </div>

        <fieldset>
          <legend className={labelClass}>{t('closeToExpiry.dialog.pricing')}</legend>
          <div role="radiogroup" className="grid grid-cols-3 gap-1 p-1 rounded-lg bg-fill">
            {PRICE_MODES.map(m => (
              <button key={m} type="button" role="radio" aria-checked={mode === m}
                onClick={() => { setMode(m); setValue(''); setError('') }}
                className={clsx('px-2 py-1.5 rounded-md text-xs font-medium transition-colors',
                  mode === m ? 'bg-raised text-ink shadow-sm' : 'text-ink-3 hover:text-ink')}>
                {t(`closeToExpiry.dialog.${MODE_LABEL[m]}`)}
              </button>
            ))}
          </div>
        </fieldset>

        <div>
          <label htmlFor="clearance-value" className={labelClass}>{t(`closeToExpiry.dialog.${VALUE_LABEL[mode]}`)}</label>
          <input id="clearance-value" inputMode="decimal" autoComplete="off" value={value}
            onChange={e => setValue(e.target.value)} className={clsx(inputClass, 'tabular-nums')} />
          {line && (
            <p className="text-xs text-ink-3 mt-1.5 tabular-nums">
              {t('closeToExpiry.dialog.preview', { price: money(previewSalePrice(batch.price_minor_units, line)), regular: money(batch.price_minor_units) })}
            </p>
          )}
        </div>

        <div>
          <label htmlFor="clearance-ends" className={labelClass}>{t('closeToExpiry.dialog.ends')}</label>
          <input id="clearance-ends" type="date" value={endsOn} onChange={e => setEndsOn(e.target.value)} className={inputClass} />
          <p className="text-2xs text-ink-4 mt-1">{t('closeToExpiry.dialog.endsHint')}</p>
        </div>

        <div>
          <label htmlFor="clearance-name" className={labelClass}>{t('closeToExpiry.dialog.name')}</label>
          <input id="clearance-name" value={name} onChange={e => setName(e.target.value)} className={inputClass} />
        </div>

        {error && <p role="alert" className="text-xs text-danger">{error}</p>}

        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} disabled={submitting}
            className="px-3 py-2 rounded-lg text-xs font-medium text-ink-3 hover:bg-hover">
            {t('closeToExpiry.dialog.cancel')}
          </button>
          <button type="submit" disabled={submitting}
            className="px-4 py-2 rounded-lg text-xs font-semibold bg-accent-solid text-white hover:bg-accent-solid-hover disabled:opacity-60">
            {submitting ? t('closeToExpiry.dialog.starting') : t('closeToExpiry.dialog.start')}
          </button>
        </div>
      </form>
    </Dialog>
  )
}

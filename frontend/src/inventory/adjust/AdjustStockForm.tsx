import { useId, useState, type ReactNode } from 'react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { Coupon } from '../../components/common/Coupon'
import { FilterChips } from '../../components/common/FilterChips'
import { apiErrorText } from '../../api/apiErrorText'
import { ExpiryStamp } from '../ExpiryStamp'
import type { RecordStockMovementInput } from '../types'
import type { StockUnit } from '../stockUnit'
import type { StockSlot } from './stockSlots'
import {
  adjustmentPreview, buildAdjustment, reasonsFor,
  type AdjustDirection, type AdjustReason, type AdjustmentPreview,
} from './adjustment'

interface AdjustStockFormProps {
  slots: StockSlot[]
  unit: StockUnit
  // A stored quantity as shown to people (kg for a weight pool).
  formatQuantity: (stored: number) => string
  onSave: (input: RecordStockMovementInput) => Promise<unknown>
  // After the server took it: the form clears for the next change.
  onRecorded: (change: AdjustmentPreview & { slot: StockSlot }) => void
  initialKey?: string
  // Opened from one batch row: only that place, no picker.
  locked?: boolean
  // 'page' lays the places out as coupons; 'compact' as a short list.
  variant?: 'page' | 'compact'
  onWriteOff?: () => void
  // Extra buttons left of "Record change" (Cancel, Open full page).
  actions?: ReactNode
}

// Adjust one SKU's stock: pick where (warehouse + lot), add or remove, how
// much, and why. The rules live in ./adjustment; this is the form around
// them, shared by the stock page and the modal over the inventory list.
export function AdjustStockForm({
  slots, unit, formatQuantity, onSave, onRecorded, initialKey, locked = false, variant = 'compact', onWriteOff, actions,
}: AdjustStockFormProps) {
  const { t } = useTranslation(['inventory', 'common'])
  // Until the person picks a place, the default follows the slots as they
  // load: the given row, else the first place with stock.
  const [chosenKey, setSelectedKey] = useState<string | null>(null)
  const selectedKey = chosenKey ?? initialKey ?? (slots.find(s => s.quantity > 0) ?? slots[0])?.key ?? ''
  const [direction, setDirection] = useState<AdjustDirection>('add')
  const [reason, setReason] = useState<AdjustReason>('received')
  const [quantity, setQuantity] = useState('')
  const [note, setNote] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const quantityId = useId()
  const noteId = useId()

  const selected = slots.find(s => s.key === selectedKey) ?? null
  const preview = selected && adjustmentPreview({ onHand: selected.quantity, direction, quantity, unit })

  const changeDirection = (next: AdjustDirection) => {
    setDirection(next)
    if (!reasonsFor(next).includes(reason)) setReason(reasonsFor(next)[0])
  }

  const submit = async () => {
    const built = buildAdjustment({
      warehouseId: selected?.warehouseId ?? '', batchId: selected?.batchId ?? '',
      direction, quantity, unit, reason, reasonLabel: t(`adjust.reason.${reason}`), note,
    })
    if (!built.ok) {
      setError(t(`adjust.error.${built.error}`))
      return
    }
    setSaving(true)
    setError(null)
    try {
      await onSave(built.input)
      if (preview && selected) onRecorded({ ...preview, slot: selected })
      setQuantity('')
      setNote('')
    } catch (e) {
      setError(apiErrorText(e) ?? t('adjust.error.failed'))
    } finally {
      setSaving(false)
    }
  }

  const page = variant === 'page'

  return (
    <form
      noValidate
      onSubmit={e => { e.preventDefault(); void submit() }}
      className={clsx('flex flex-col', page ? 'gap-6' : 'gap-4')}
    >
      {locked && selected
        ? <LockedPlace slot={selected} formatQuantity={formatQuantity} />
        : <PlacePicker slots={slots} selectedKey={selectedKey} onSelect={setSelectedKey} formatQuantity={formatQuantity} page={page} />}

      <div className={clsx('flex flex-wrap gap-x-6 gap-y-3', page && 'items-end')}>
        <FilterChips
          label={t('adjust.change')}
          options={[{ value: 'add', label: t('adjust.add') }, { value: 'remove', label: t('adjust.remove') }]}
          selected={[direction]}
          onToggle={changeDirection}
        />
        <FilterChips
          label={t('adjust.why')}
          options={reasonsFor(direction).map(r => ({ value: r, label: t(`adjust.reason.${r}`) }))}
          selected={[reason]}
          onToggle={setReason}
        />
      </div>

      <div className={clsx('grid gap-3', page ? 'sm:grid-cols-[12rem_1fr]' : 'grid-cols-[7rem_1fr]')}>
        <div className="flex flex-col gap-1">
          <label htmlFor={quantityId} className="caps-label-widest text-2xs font-semibold text-ink-4">
            {unit.kind === 'kg' ? t('adjust.quantityKg') : t('adjust.quantity')}
          </label>
          <input
            id={quantityId}
            type="number"
            min={0}
            step={unit.step}
            inputMode="decimal"
            value={quantity}
            onChange={e => setQuantity(e.target.value)}
            className={clsx(
              'w-full rounded-md border border-line bg-surface px-2.5 text-ink tabular-nums outline-none focus:ring-2 focus:ring-focus',
              page ? 'h-11 text-xl font-bold' : 'h-8 text-sm font-semibold',
            )}
          />
        </div>
        <div className="flex flex-col gap-1">
          <label htmlFor={noteId} className="caps-label-widest text-2xs font-semibold text-ink-4">{t('adjust.note')}</label>
          <input
            id={noteId}
            value={note}
            onChange={e => setNote(e.target.value)}
            placeholder={t('adjust.notePlaceholder')}
            className={clsx(
              'w-full rounded-md border border-line bg-surface px-2.5 text-xs text-ink placeholder-ink-4 outline-none focus:ring-2 focus:ring-focus',
              page ? 'h-11' : 'h-8',
            )}
          />
        </div>
      </div>

      {direction === 'remove' && onWriteOff && (
        <p className="text-xs text-ink-3">
          {t('adjust.writeOffHint')}{' '}
          <button type="button" onClick={onWriteOff} className="font-semibold text-accent underline-offset-2 hover:underline">
            {t('adjust.writeOff')}
          </button>
        </p>
      )}

      <div className={clsx('flex flex-wrap items-center gap-3 border-t border-line pt-4', page && 'justify-between')}>
        <Preview preview={preview} slot={selected} formatQuantity={formatQuantity} />
        <div className="ml-auto flex items-center gap-2">
          {actions}
          <button
            type="submit"
            disabled={saving || !!preview?.short}
            className="h-8 rounded-md bg-accent-solid px-3.5 text-xs font-semibold text-white hover:bg-accent-solid-hover disabled:opacity-50"
          >
            {saving ? t('common:actions.saving') : t('adjust.save')}
          </button>
        </div>
      </div>
      {error && <p role="alert" className="-mt-2 text-xs text-danger">{error}</p>}
    </form>
  )
}

// Long figures (kg with decimals) step down so they stay inside the stub.
function StubFigure({ value }: { value: string }) {
  return <span className={value.length > 4 ? 'text-lg' : undefined}>{value}</span>
}

// "No lot" or "Lot L-2411".
function usePlaceName(): (slot: StockSlot) => string {
  const { t } = useTranslation('inventory')
  return slot => (slot.isLedger ? t('adjust.noLot') : t('adjust.lot', { lot: slot.lot || '—' }))
}

function Preview({ preview, slot, formatQuantity }: {
  preview: AdjustmentPreview | null; slot: StockSlot | null; formatQuantity: (n: number) => string
}) {
  const { t } = useTranslation('inventory')
  const placeName = usePlaceName()
  if (!preview || !slot) return <span className="text-xs text-ink-4">{t('adjust.previewLabel')}: —</span>
  return (
    <div className="flex flex-col gap-0.5">
      <span className="caps-label-widest text-2xs font-semibold text-ink-4">
        {t('adjust.previewLabel')} · {slot.warehouseName} · {placeName(slot)}
      </span>
      <span data-testid="adjust-preview" className="text-lg font-bold tabular-nums text-figure">
        {formatQuantity(preview.before)} → <span className={preview.short ? 'text-danger' : undefined}>{formatQuantity(preview.after)}</span>
      </span>
      {preview.short && <span className="text-xs font-medium text-danger">{t('adjust.short', { onHand: formatQuantity(preview.before) })}</span>}
    </div>
  )
}

function LockedPlace({ slot, formatQuantity }: { slot: StockSlot; formatQuantity: (n: number) => string }) {
  const { t } = useTranslation('inventory')
  return (
    <div data-testid="adjust-where" className="flex items-center justify-between gap-3 rounded-md border border-line bg-sunken px-3 py-2">
      <div className="min-w-0">
        <div className="caps-label-widest text-2xs font-semibold text-ink-4">{t('adjust.where')}</div>
        <div className="truncate text-xs font-semibold text-ink">
          <span>{slot.warehouseName}</span>
          <span className="text-ink-4"> · </span>
          {slot.isLedger ? <span>{t('adjust.noLot')}</span> : <><span className="text-ink-3">{t('adjust.lotWord')} </span><span>{slot.lot || '—'}</span></>}
        </div>
      </div>
      <span className="text-sm font-bold tabular-nums text-figure">{formatQuantity(slot.quantity)}</span>
    </div>
  )
}

function PlacePicker({ slots, selectedKey, onSelect, formatQuantity, page }: {
  slots: StockSlot[]; selectedKey: string; onSelect: (key: string) => void; formatQuantity: (n: number) => string; page: boolean
}) {
  const { t } = useTranslation('inventory')
  const placeName = usePlaceName()
  const [showEmpty, setShowEmpty] = useState(false)
  const name = useId()
  const labelId = useId()
  const emptyCount = slots.filter(s => s.empty).length
  const visible = slots.filter(s => !s.empty || showEmpty || s.key === selectedKey)

  return (
    <div role="radiogroup" aria-labelledby={labelId} className="flex flex-col gap-2">
      <span id={labelId} className="caps-label-widest text-2xs font-semibold text-ink-4">{t('adjust.where')}</span>
      <div className={clsx(page ? 'grid gap-3 sm:grid-cols-2 xl:grid-cols-3' : 'flex flex-col gap-1.5 max-h-56 overflow-y-auto')}>
        {visible.map(slot => {
          const on = slot.key === selectedKey
          const radio = (
            <input
              type="radio"
              name={name}
              value={slot.key}
              checked={on}
              onChange={() => onSelect(slot.key)}
              className="sr-only"
            />
          )
          if (page) {
            return (
              <label key={slot.key} className="block cursor-pointer rounded-lg has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-focus">
                {radio}
                <Coupon tone={on ? 'raised' : slot.empty ? 'waiting' : 'plain'} stubLabel={t('adjust.onHand')} stubValue={<StubFigure value={formatQuantity(slot.quantity)} />} stubWidth="wide">
                  <span className="text-sm font-semibold text-ink">{placeName(slot)}</span>
                  <span className="flex flex-wrap items-center gap-1.5 text-xs text-ink-3">
                    {slot.warehouseName}
                    <ExpiryStamp date={slot.expiryDate} />
                  </span>
                </Coupon>
              </label>
            )
          }
          return (
            <label
              key={slot.key}
              className={clsx(
                'flex cursor-pointer items-center gap-2 rounded-md border px-2.5 py-1.5 text-xs has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-focus',
                on ? 'border-accent bg-accent/10' : 'border-line hover:bg-hover',
              )}
            >
              {radio}
              <span className={clsx('h-2.5 w-2.5 shrink-0 rounded-full border', on ? 'border-accent bg-accent' : 'border-ink-4')} aria-hidden />
              <span className="min-w-0 flex-1 truncate">
                <span className="font-semibold text-ink">{placeName(slot)}</span>
                <span className="text-ink-3"> · {slot.warehouseName}</span>
              </span>
              <ExpiryStamp date={slot.expiryDate} />
              <span className="font-bold tabular-nums text-figure">{formatQuantity(slot.quantity)}</span>
            </label>
          )
        })}
      </div>
      {emptyCount > 0 && (
        <button type="button" onClick={() => setShowEmpty(v => !v)} className="self-start text-2xs font-medium text-ink-4 hover:text-accent">
          {showEmpty ? t('adjust.hideEmpty') : t('adjust.showEmpty', { count: emptyCount })}
        </button>
      )}
    </div>
  )
}

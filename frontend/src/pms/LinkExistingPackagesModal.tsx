import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Modal } from '../inventory/components/Modal'
import { Money, useMoneyFormat } from '../format/amountVisibility'
import * as pmsApi from './api'
import type { LinkExistingWeightPackagesResult } from './types'
import { apiErrorText } from '../api/apiErrorText'

// Adopts packages made by hand (variants carrying this pool's scale label)
// as the pool's packages -- previews first (dry run), links on confirm.
// See backend application/pms.LinkExistingWeightPackagesUseCase.
export function LinkExistingPackagesModal({ token, bulkSkuId, currencyCode, onLinked, onClose }: {
  token: string
  bulkSkuId: string
  currencyCode: string
  onLinked: () => void
  onClose: () => void
}) {
  const { t } = useTranslation(['pms', 'common'])
  const formatMoney = useMoneyFormat()
  const [preview, setPreview] = useState<LinkExistingWeightPackagesResult | null>(null)
  const [done, setDone] = useState<LinkExistingWeightPackagesResult | null>(null)
  const [linking, setLinking] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    pmsApi.linkExistingWeightPackages(token, bulkSkuId, true)
      .then(r => { if (!cancelled) setPreview(r) })
      .catch(e => { if (!cancelled) setError(apiErrorText(e) ?? t('split.linkFailed')) })
    return () => { cancelled = true }
  }, [token, bulkSkuId, t])

  async function link() {
    setLinking(true)
    setError('')
    try {
      const result = await pmsApi.linkExistingWeightPackages(token, bulkSkuId, false)
      setDone(result)
      onLinked()
    } catch (e) {
      setError(apiErrorText(e) ?? t('split.linkFailed'))
    } finally {
      setLinking(false)
    }
  }

  const shown = done ?? preview
  return (
    <Modal title={t('split.linkTitle')} onClose={onClose}>
      <div className="space-y-3 text-2xs">
        <p className="text-ink-4">{t('split.linkIntro')}</p>
        {!shown && !error && <p className="text-ink-3">{t('split.linkLoading')}</p>}
        {done && <p className="text-success">{t('split.linkDone', { count: done.linked.length })}</p>}
        {shown && shown.linked.length === 0 && !done && <p className="text-ink-3">{t('split.linkNone')}</p>}
        {shown && shown.linked.length > 0 && (
          <ul className="rounded-lg border border-line divide-y divide-line max-h-56 overflow-y-auto">
            {shown.linked.map(p => (
              <li key={p.sku_id} className="flex items-center justify-between gap-2 px-2 py-1">
                <span className="text-ink-2">{p.variant_name} <span className="text-ink-4">{p.sku_code}</span></span>
                <span className="text-ink-3">
                  {p.price_changed ? t('split.linkPriceChanged', { price: formatMoney(p.price_minor_units, currencyCode) }) : <Money minor={p.price_minor_units} currency={currencyCode} />}
                </span>
              </li>
            ))}
          </ul>
        )}
        {shown && shown.skipped.length > 0 && (
          <div>
            <p className="text-ink-3 mb-1">{t('split.linkSkippedTitle')}</p>
            <ul className="space-y-0.5">
              {shown.skipped.map(s => (
                <li key={s.sku_id} className="flex justify-between gap-2 text-ink-4">
                  <span>{s.variant_name}</span><span>{t(`split.linkSkip.${s.reason}`)}</span>
                </li>
              ))}
            </ul>
          </div>
        )}
        {done && done.errors.map((e, i) => <p key={i} className="text-danger">{e}</p>)}
        {error && <p className="text-danger">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg text-ink-2 hover:bg-hover">
            {done ? t('common:actions.close') : t('common:actions.cancel')}
          </button>
          {!done && preview && preview.linked.length > 0 && (
            <button
              type="button" onClick={link} disabled={linking}
              className="px-3 py-1.5 text-xs rounded-lg bg-accent-solid text-white hover:bg-accent-solid-hover disabled:opacity-50"
            >
              {linking ? t('split.linking') : t('split.linkConfirm', { count: preview.linked.length })}
            </button>
          )}
        </div>
      </div>
    </Modal>
  )
}

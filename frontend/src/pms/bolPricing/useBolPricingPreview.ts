import i18next from 'i18next'
import { useEffect, useRef, useState } from 'react'
import { apiErrorText } from '../../api/apiErrorText'
import * as integrationsApi from '../../integrations/api'
import type { BolPricingDraft, BolPricingPreview } from '../../integrations/types'

const PREVIEW_DELAY_MS = 400

export interface BolPricingPreviewState {
  preview: BolPricingPreview | null
  loading: boolean
  error: string | null
}

// Asks the backend what a push would send to Bol for the draft settings,
// once typing pauses; a newer draft always wins over a slower answer.
export function useBolPricingPreview(
  token: string | null, connectionId: string, productId: string, skuId: string | null, draft: BolPricingDraft,
): BolPricingPreviewState {
  const [state, setState] = useState<BolPricingPreviewState>({ preview: null, loading: false, error: null })
  const latest = useRef(0)
  const key = JSON.stringify(draft)

  useEffect(() => {
    if (!token || !skuId) return
    const call = ++latest.current
    const timer = setTimeout(() => {
      setState(s => ({ ...s, loading: true }))
      integrationsApi.previewBolPricing(token, connectionId, productId, skuId, JSON.parse(key) as BolPricingDraft)
        .then(preview => { if (call === latest.current) setState({ preview, loading: false, error: null }) })
        .catch(err => {
          if (call === latest.current) {
            setState({ preview: null, loading: false, error: apiErrorText(err) ?? i18next.t('pms:bol.pricing.previewFailed') })
          }
        })
    }, PREVIEW_DELAY_MS)
    return () => clearTimeout(timer)
  }, [token, connectionId, productId, skuId, key])

  return state
}

import i18next from 'i18next'
import { useCallback, useEffect, useMemo, useState } from 'react'
import type { ProductMergePreview, ProductMergeResult, SKUMergePair } from '../types'
import * as api from '../api'
import { apiErrorText } from '../../api/apiErrorText'

// Merging a duplicate product into a chosen existing one: the server's
// preview (stock and cost after), the user's SKU matching (seeded from the
// server's suggestion), and the merge itself. Matching every duplicate
// SKU re-previews with that exact mapping, so what's shown is what merges.
export function useProductMerge(token: string, duplicateId: string, targetId: string | null) {
  const [preview, setPreview] = useState<ProductMergePreview | null>(null)
  const [mapping, setMapping] = useState<Record<string, string>>({})
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [userMatched, setUserMatched] = useState(false)

  const pairs: SKUMergePair[] = useMemo(
    () => Object.entries(mapping).map(([source_sku_id, target_sku_id]) => ({ source_sku_id, target_sku_id })),
    [mapping],
  )
  const complete = !!preview && preview.source.skus.every(s => mapping[s.id])

  const load = useCallback(async (skuMapping?: SKUMergePair[]) => {
    if (!targetId) return
    setLoading(true)
    setError(null)
    try {
      const next = await api.previewProductMerge(token, duplicateId, {
        target_product_id: targetId, ...(skuMapping ? { sku_mapping: skuMapping } : {}),
      })
      setPreview(next)
      if (!skuMapping) setMapping(Object.fromEntries(next.sku_mapping.map(p => [p.source_sku_id, p.target_sku_id])))
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('pms:errors.mergePreview'))
    } finally {
      setLoading(false)
    }
  }, [token, duplicateId, targetId])

  // A new target: start from the server's suggestion.
  useEffect(() => {
    setPreview(null)
    setMapping({})
    setUserMatched(false)
    void load()
  }, [load])

  // Once the user has matched SKUs themselves and every one is matched,
  // project exactly that mapping.
  const match = useCallback((sourceSkuId: string, targetSkuId: string) => {
    setUserMatched(true)
    setMapping(m => ({ ...m, [sourceSkuId]: targetSkuId }))
  }, [])
  useEffect(() => {
    if (userMatched && complete) void load(pairs)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- re-preview on mapping changes only
  }, [pairs])

  const merge = useCallback((): Promise<ProductMergeResult> => {
    if (!targetId) return Promise.reject(new Error(i18next.t('pms:errors.chooseMergeTarget')))
    return api.mergeProduct(token, duplicateId, { target_product_id: targetId, sku_mapping: pairs })
  }, [token, duplicateId, targetId, pairs])

  return { preview, mapping, complete, loading, error, match, merge }
}

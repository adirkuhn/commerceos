import { useEffect, useState } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import { Puck, type Data } from '@puckeditor/core'
import '@puckeditor/core/puck.css'
import { setStorefrontKey } from '../api/client'
import { getThemeDraft, saveThemeDraft } from '../api/themeDraft'
import { buildPuckConfig } from '../puck/config'
import type { PageSlug, ThemePage } from '../types'

const EMPTY_PUCK_DATA: Data = { content: [], root: {} }

// EditPage is storefront-frontend's own embedded visual (Puck) editor --
// mounted at /edit/:slug (see App.tsx), reached only from the admin app's
// "Edit visually" launcher (frontend/pages/ThemeEditorPage.tsx), which
// mints the ?token= this page authenticates every request with (see
// backend application/theme.CreateEditSessionUseCase). Never linked from
// anywhere a shopper would see -- this is a merchant tool, not part of
// the storefront.
export default function EditPage() {
  const { slug } = useParams<{ slug: string }>()
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') ?? ''
  const pageSlug = slug as PageSlug

  const [page, setPage] = useState<ThemePage | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!token) {
      setError('Missing edit session token -- open this page from the admin app\'s "Edit visually" button.')
      return
    }
    getThemeDraft(token, pageSlug)
      .then(fetchedPage => {
        // Scopes the editor's own live blocks (ProductGrid, CartSummary,
        // CheckoutForm, ...) to this edit session's channel -- api/client.ts's
        // storefront key is shared per-origin state and this same deployment
        // serves every tenant, so it must be set fresh on every load rather
        // than trusting whatever a previous shopper/merchant visit left in
        // localStorage (see StoreEntry's identical reasoning for shoppers).
        if (fetchedPage.sales_channel_id) setStorefrontKey(fetchedPage.sales_channel_id)
        setPage(fetchedPage)
      })
      .catch((err: Error) => setError(err.message))
  }, [token, pageSlug])

  if (error) {
    return <p className="p-6 text-sm text-red-600">Could not load this page: {error}</p>
  }
  if (!page) {
    return <p className="p-6 text-sm text-gray-500">Loading…</p>
  }

  const data = (page.puck_data as Data | undefined) ?? EMPTY_PUCK_DATA

  return (
    <Puck
      config={buildPuckConfig(pageSlug)}
      data={data}
      onPublish={async (newData: Data) => {
        // Round-trips page's own freeform blocks/custom_css/canvas_width
        // unchanged -- a Puck save must never clobber the freeform
        // document, only switch which one is active (see
        // domain/theme.ThemePage's coexisting-documents doc comment).
        await saveThemeDraft(token, pageSlug, {
          canvas_width: page.canvas_width,
          blocks: page.blocks,
          custom_css: page.custom_css,
          editor_mode: 'puck',
          puck_data: newData as Record<string, unknown>,
        })
        setPage({ ...page, editor_mode: 'puck', puck_data: newData as Record<string, unknown> })
      }}
    />
  )
}

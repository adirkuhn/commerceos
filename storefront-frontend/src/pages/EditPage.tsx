import { useEffect, useState } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import { Puck, usePuck, type Data } from '@puckeditor/core'
import '@puckeditor/core/puck.css'
import { CartProvider } from '../cart/CartContext'
import { WishlistProvider } from '../wishlist/WishlistContext'
import { AuthProvider } from '../auth/AuthContext'
import { setStorefrontKey } from '../api/client'
import { listCategories } from '../api/categories'
import { getThemeDraft, saveThemeDraft } from '../api/themeDraft'
import { buildPuckConfig } from '../puck/config'
import type { PageSlug, StorefrontCategory, ThemePage } from '../types'

const EMPTY_PUCK_DATA: Data = { content: [], root: {} }

// SaveButton reads Puck's own live in-progress data via usePuck() (only
// callable from a component Puck itself renders, e.g. via the
// headerActions override below) -- lets a merchant persist their draft
// without also flipping editor_mode to 'puck' (see EditPage's own
// onSave/onPublish split for why that distinction matters).
function SaveButton({ onSave }: { onSave: (data: Data) => void }) {
  const { appState } = usePuck()
  return (
    <button
      type="button"
      onClick={() => onSave(appState.data)}
      className="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50"
    >
      Save
    </button>
  )
}

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
  const [categories, setCategories] = useState<StorefrontCategory[]>([])
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
        // Must happen before listCategories below -- that call rides the
        // same shared key.
        if (fetchedPage.sales_channel_id) setStorefrontKey(fetchedPage.sales_channel_id)
        setPage(fetchedPage)
        // Best-effort: only powers FeaturedProducts' category dropdown,
        // which still works (as "All categories") if this fails.
        listCategories().then(setCategories).catch(() => {})
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

  // Round-trips page's own freeform blocks/custom_css/canvas_width
  // unchanged either way -- a Puck save must never clobber the freeform
  // document, only (optionally) switch which one is active (see
  // domain/theme.ThemePage's coexisting-documents doc comment). Save
  // keeps whatever editor_mode the page already had (a merchant
  // persisting in-progress Puck work without making it the live one);
  // Publish is the only action that sets it to 'puck'.
  const persist = async (newData: Data, publish: boolean) => {
    const editorMode = publish ? 'puck' : page.editor_mode
    await saveThemeDraft(token, pageSlug, {
      canvas_width: page.canvas_width,
      blocks: page.blocks,
      custom_css: page.custom_css,
      editor_mode: editorMode,
      puck_data: newData as Record<string, unknown>,
    })
    setPage({ ...page, editor_mode: editorMode, puck_data: newData as Record<string, unknown> })
  }

  return (
    // The editor canvas renders the real, live functional blocks
    // (NavActionsBlock, CartSummaryBlock, CheckoutFormBlock, WishlistBlock,
    // LoginFormBlock, AccountBlock) -- they all need real cart/wishlist/
    // auth context, same as a shopper's own ShopperRoutes (see App.tsx).
    // Without these, any page containing one of them throws
    // "useX must be used within a XProvider" as soon as it renders.
    <CartProvider>
      <WishlistProvider>
        <AuthProvider>
          <Puck
            config={buildPuckConfig(pageSlug, categories)}
            data={data}
            onPublish={(newData: Data) => persist(newData, true)}
            overrides={{
              headerActions: ({ children }) => (
                <>
                  <SaveButton onSave={newData => persist(newData, false)} />
                  {children}
                </>
              ),
            }}
          />
        </AuthProvider>
      </WishlistProvider>
    </CartProvider>
  )
}

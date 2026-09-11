import { useEffect, useRef, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { ArrowLeft, Code, ExternalLink, Wand2 } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useThemePage } from '../theme/hooks/useThemePage'
import { themeEditorParamsFromPath, themeEditorPath } from '../theme/routing'
import { blocksToPuckData } from '../theme/puckConversion'
import { createEditSession } from '../theme/api'
import { storefrontEditURL, storefrontPreviewURL } from '../theme/storefrontLink'
import { CustomCSSPanel } from '../theme/components/CustomCSSPanel'
import { PAGE_SLUGS, PAGE_SLUG_LABELS } from '../theme/types'
import type { PageSlug } from '../theme/types'

// Freeform (the old drag/resize canvas -- BlockPreview, BlockPropsPanel,
// the Rnd-based layout editor) is gone. Puck (storefront-frontend's own
// embedded visual editor, see openVisualEditor below) is now the only way
// to edit a page's content -- it already registers every block type as a
// real, live, wired-up component (see storefront-frontend/puck/config.tsx),
// unlike the old canvas's own rough approximation. This page now only
// handles page selection, launching Puck, previewing the live result, and
// custom_css -- the one thing Puck itself has no UI for.
export default function ThemeEditorPage() {
  const { token } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const params = themeEditorParamsFromPath(location.pathname)

  if (!params) {
    return <InvalidRoute />
  }
  return <ThemeEditor token={token ?? ''} channelId={params.channelId} slug={params.slug} navigate={navigate} />
}

function InvalidRoute() {
  return (
    <div className="flex h-full items-center justify-center">
      <p className="text-sm text-gray-500">This theme page URL isn&apos;t valid.</p>
    </div>
  )
}

interface EditorProps {
  token: string
  channelId: string
  slug: PageSlug
  navigate: ReturnType<typeof useNavigate>
}

function ThemeEditor({ token, channelId, slug, navigate }: EditorProps) {
  const { page, loading, saving, error, save } = useThemePage(token, channelId, slug)

  const [customCss, setCustomCss] = useState('')
  const [cssDirty, setCssDirty] = useState(false)
  const [cssPanelOpen, setCssPanelOpen] = useState(false)
  const [editSessionLoading, setEditSessionLoading] = useState(false)
  const [editSessionError, setEditSessionError] = useState<string | null>(null)
  const initializedFor = useRef<string | null>(null)

  useEffect(() => {
    const key = `${channelId}|${slug}`
    if (page && initializedFor.current !== key) {
      setCustomCss(page.custom_css)
      setCssDirty(false)
      setCssPanelOpen(false)
      initializedFor.current = key
      // A page created before straight-to-Puck seeding (see
      // SalesChannels.tsx's seedStarterTheme) or the "Migrate to Puck"
      // bulk action still has editor_mode 'freeform' with real blocks and
      // empty puck_data -- opening it here converts it in place, the same
      // one-page version of that bulk action's own per-page logic, so
      // simply opening any page (even just to edit its Custom CSS) is
      // never blocked on a merchant remembering to migrate it first.
      if (page.editor_mode === 'freeform' && page.blocks.length > 0) {
        save({
          canvas_width: page.canvas_width,
          blocks: page.blocks,
          custom_css: page.custom_css,
          editor_mode: 'puck',
          puck_data: blocksToPuckData(page.blocks),
        })
      }
    }
    // save is stable across renders (see useThemePage) but is
    // deliberately omitted -- including it would re-run this effect on
    // every save, including the very auto-migration save it triggers.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, channelId, slug])

  const openCssPanel = () => setCssPanelOpen(o => !o)

  const handleSaveCss = async () => {
    if (!page) return
    await save({
      canvas_width: page.canvas_width,
      blocks: page.blocks,
      custom_css: customCss,
      editor_mode: 'puck',
      puck_data: page.puck_data,
    })
    setCssDirty(false)
  }

  // openVisualEditor mints a short-lived edit session token (a distinct,
  // higher-privilege credential from any storefront key -- see backend
  // application/theme.CreateEditSessionUseCase's own doc comment) and
  // opens storefront-frontend's embedded Puck editor in a new tab, never
  // an iframe: the editor needs the real live storefront components and
  // styling, which only storefront-frontend itself can render.
  const openVisualEditor = async () => {
    setEditSessionError(null)
    setEditSessionLoading(true)
    try {
      const session = await createEditSession(token, channelId)
      const url = storefrontEditURL(slug, session.token)
      if (!url) {
        setEditSessionError('Storefront URL is not configured.')
        return
      }
      window.open(url, '_blank', 'noopener,noreferrer')
    } catch (e) {
      setEditSessionError(e instanceof Error ? e.message : 'Could not start a visual edit session.')
    } finally {
      setEditSessionLoading(false)
    }
  }

  // The real, saved, live-rendered page -- see storefrontPreviewURL's own
  // doc comment for which slugs this covers. Always the last-saved
  // version (unsaved custom_css edits aren't reflected until Save).
  const previewURL = storefrontPreviewURL(channelId, slug)

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="flex shrink-0 items-center justify-between border-b border-gray-200 bg-gray-50 px-6 py-4">
        <div>
          <button
            onClick={() => navigate('/sales-channels')}
            className="mb-1 flex items-center gap-1 text-[11px] font-medium text-gray-400 transition-colors hover:text-gray-600"
          >
            <ArrowLeft size={11} /> Sales channels
          </button>
          <div className="flex items-center gap-3">
            <h1 className="text-xl font-semibold leading-tight text-gray-900">Theme editor</h1>
            <select
              aria-label="Page"
              value={slug}
              onChange={e => navigate(themeEditorPath(channelId, e.target.value as PageSlug))}
              className="rounded-lg border border-gray-200 bg-white px-2 py-1 text-xs text-gray-700 outline-none"
            >
              {PAGE_SLUGS.map(s => <option key={s} value={s}>{PAGE_SLUG_LABELS[s]}</option>)}
            </select>
          </div>
        </div>
        <div className="flex items-center gap-3">
          {error && <p className="text-xs text-red-500">{error}</p>}
          {editSessionError && <p className="text-xs text-red-500">{editSessionError}</p>}
          <button
            onClick={openVisualEditor}
            disabled={editSessionLoading}
            title="Opens storefront-frontend's embedded visual editor in a new tab -- the real live components, true WYSIWYG"
            className="flex items-center gap-1.5 rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-xs font-medium text-gray-600 transition-colors hover:bg-gray-50 disabled:opacity-50"
          >
            <Wand2 size={12} /> {editSessionLoading ? 'Opening…' : 'Edit visually'}
          </button>
          {previewURL ? (
            <a
              href={previewURL}
              target="_blank"
              rel="noreferrer"
              title="Opens the real, saved, live-rendered page in a new tab"
              className="flex items-center gap-1.5 rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-xs font-medium text-gray-600 transition-colors hover:bg-gray-50"
            >
              <ExternalLink size={12} /> Preview
            </a>
          ) : (
            <button
              disabled
              title="This page has no fixed storefront route to preview generically (e.g. it needs a real product/order first)"
              className="flex items-center gap-1.5 rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-xs font-medium text-gray-300"
            >
              <ExternalLink size={12} /> Preview
            </button>
          )}
          <button
            onClick={openCssPanel}
            className={clsx(
              'flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors',
              cssPanelOpen ? 'border-brand-600 bg-brand-50 text-brand-700' : 'border-gray-200 bg-white text-gray-600 hover:bg-gray-50',
            )}
          >
            <Code size={12} /> Custom CSS
          </button>
          <button
            onClick={handleSaveCss}
            disabled={saving || !cssDirty}
            className="rounded-lg bg-brand-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-brand-700 disabled:opacity-50"
          >
            {saving ? 'Saving…' : 'Save'}
          </button>
        </div>
      </div>

      <div className="flex flex-1 overflow-hidden">
        <div className="flex-1 overflow-auto bg-gray-100 p-8">
          {loading ? (
            <p className="text-sm text-gray-500">Loading…</p>
          ) : (
            <div className="mx-auto max-w-xl rounded-xl border border-dashed border-gray-300 bg-white p-8 text-center">
              <Wand2 size={20} className="mx-auto mb-2 text-gray-400" />
              <p className="text-sm font-medium text-gray-700">Content is edited in Puck</p>
              <p className="mt-1 text-xs text-gray-500">
                Click "Edit visually" above to open storefront-frontend's own embedded editor -- the real, live
                components, not an approximation. Custom CSS (colors, typography) is the one thing edited here.
              </p>
            </div>
          )}
        </div>

        {cssPanelOpen && (
          <div className="w-80 shrink-0 overflow-y-auto border-l border-gray-200 bg-white">
            <CustomCSSPanel value={customCss} onChange={v => { setCustomCss(v); setCssDirty(true) }} />
          </div>
        )}
      </div>
    </div>
  )
}

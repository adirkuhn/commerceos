import { useEffect, useRef, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { Rnd } from 'react-rnd'
import { ArrowLeft, Plus, Code, Monitor, Smartphone, ExternalLink, Wand2 } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useThemePage } from '../theme/hooks/useThemePage'
import { themeEditorParamsFromPath, themeEditorPath } from '../theme/routing'
import { createBlock } from '../theme/blockDefaults'
import { THEME_PALETTES, paletteCustomCSS } from '../theme/palettes'
import { STARTER_TEMPLATES, combinedStarterCSS } from '../theme/starterTemplate'
import { scopeCssToCanvas } from '../theme/scopeCanvasCss'
import type { StarterTemplateId } from '../theme/starterTemplate'
import { createEditSession } from '../theme/api'
import { storefrontEditURL, storefrontPreviewURL } from '../theme/storefrontLink'
import { defaultMobileRect, MOBILE_CANVAS_WIDTH } from '../theme/mobileLayout'
import { BlockPreview } from '../theme/components/BlockPreview'
import { BlockPropsPanel } from '../theme/components/BlockPropsPanel'
import { CustomCSSPanel } from '../theme/components/CustomCSSPanel'
import { PAGE_SLUGS, PAGE_SLUG_LABELS, EDITABLE_BLOCK_TYPES, BLOCK_TYPE_LABELS } from '../theme/types'
import type { Block, EditorMode, PageSlug, Rect } from '../theme/types'

const CANVAS_HEIGHT = 900
type Breakpoint = 'desktop' | 'mobile'

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

  const [blocks, setBlocks] = useState<Block[]>([])
  const [canvasWidth, setCanvasWidth] = useState(1440)
  const [customCss, setCustomCss] = useState('')
  const [breakpoint, setBreakpoint] = useState<Breakpoint>('desktop')
  const [selectedBlockId, setSelectedBlockId] = useState<string | null>(null)
  const [cssPanelOpen, setCssPanelOpen] = useState(false)
  const [addMenuOpen, setAddMenuOpen] = useState(false)
  const [templatesOpen, setTemplatesOpen] = useState(false)
  const [selectedTemplateId, setSelectedTemplateId] = useState<StarterTemplateId>('classic')
  const [dirty, setDirty] = useState(false)
  const [editSessionLoading, setEditSessionLoading] = useState(false)
  const [editSessionError, setEditSessionError] = useState<string | null>(null)
  const initializedFor = useRef<string | null>(null)

  useEffect(() => {
    const key = `${channelId}|${slug}`
    if (page && initializedFor.current !== key) {
      setBlocks(page.blocks)
      setCanvasWidth(page.canvas_width)
      setCustomCss(page.custom_css)
      setSelectedBlockId(null)
      setCssPanelOpen(false)
      setBreakpoint('desktop')
      setDirty(false)
      initializedFor.current = key
    }
  }, [page, channelId, slug])

  const selectedBlock = blocks.find(b => b.id === selectedBlockId) ?? null

  const selectBlock = (id: string | null) => {
    setSelectedBlockId(id)
    if (id) setCssPanelOpen(false)
  }

  const openCssPanel = () => {
    setCssPanelOpen(true)
    setSelectedBlockId(null)
  }

  const updateBlock = (id: string, updater: (b: Block) => Block) => {
    setBlocks(prev => prev.map(b => (b.id === id ? updater(b) : b)))
    setDirty(true)
  }

  // The Rect the canvas actually displays/edits for the current
  // breakpoint -- Mobile falls back to a computed full-width default
  // (see mobileLayout.ts) until a merchant's first drag/resize in Mobile
  // view gives it a real saved override.
  const rectFor = (block: Block): Rect => (breakpoint === 'desktop' ? block.layout.desktop : block.layout.mobile ?? defaultMobileRect(block.layout.desktop))

  const updateRect = (id: string, rect: Rect) => {
    updateBlock(id, b => ({
      ...b,
      layout: breakpoint === 'desktop' ? { ...b.layout, desktop: rect } : { ...b.layout, mobile: rect },
    }))
  }

  const addBlock = (type: (typeof EDITABLE_BLOCK_TYPES)[number]) => {
    const block = createBlock(type)
    setBlocks(prev => [...prev, block])
    selectBlock(block.id)
    setDirty(true)
    setAddMenuOpen(false)
  }

  const deleteBlock = (id: string) => {
    setBlocks(prev => prev.filter(b => b.id !== id))
    if (selectedBlockId === id) selectBlock(null)
    setDirty(true)
  }

  const applyTemplate = (palette: (typeof THEME_PALETTES)[number]) => {
    if (blocks.length > 0 && !window.confirm('Replace this page\'s current blocks with the starter template?')) {
      return
    }
    const template = STARTER_TEMPLATES.find(t => t.id === selectedTemplateId) ?? STARTER_TEMPLATES[0]
    setBlocks(template.buildHome(''))
    setCustomCss(combinedStarterCSS(paletteCustomCSS(palette), selectedTemplateId))
    selectBlock(null)
    setCssPanelOpen(false)
    setDirty(true)
    setTemplatesOpen(false)
  }

  // editorMode is which document the storefront actually renders live --
  // 'freeform' (this canvas) or 'puck' (storefront-frontend's own
  // embedded Puck editor, see openVisualEditor). Saving this canvas never
  // changes it implicitly: a merchant polishing freeform content while
  // Puck is live must not accidentally flip the storefront back. Only
  // switchMode changes it, explicitly, round-tripping whichever document
  // isn't being switched to unchanged (see backend domain/theme.ThemePage's
  // coexisting-documents doc comment).
  const editorMode: EditorMode = page?.editor_mode ?? 'freeform'

  const handleSave = async () => {
    await save({ canvas_width: canvasWidth, blocks, custom_css: customCss, editor_mode: editorMode, puck_data: page?.puck_data })
    setDirty(false)
  }

  const switchMode = async (mode: EditorMode) => {
    if (mode === editorMode) return
    await save({ canvas_width: canvasWidth, blocks, custom_css: customCss, editor_mode: mode, puck_data: page?.puck_data })
    setDirty(false)
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

  const displayedWidth = breakpoint === 'desktop' ? canvasWidth : MOBILE_CANVAS_WIDTH
  // The real, saved, live-rendered page -- see storefrontPreviewURL's own
  // doc comment for which slugs this covers. Always the last-saved
  // version (unsaved edits in this canvas aren't reflected until Save).
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
          <div className="flex items-center rounded-lg border border-gray-200 bg-white p-0.5">
            <button
              aria-pressed={editorMode === 'freeform'}
              onClick={() => switchMode('freeform')}
              title="Storefront renders this canvas's blocks live"
              className={clsx('rounded-md px-2 py-1 text-xs font-medium transition-colors', editorMode === 'freeform' ? 'bg-brand-600 text-white' : 'text-gray-400 hover:text-gray-600')}
            >
              Freeform
            </button>
            <button
              aria-pressed={editorMode === 'puck'}
              onClick={() => switchMode('puck')}
              title="Storefront renders the visual (Puck) editor's content live"
              className={clsx('rounded-md px-2 py-1 text-xs font-medium transition-colors', editorMode === 'puck' ? 'bg-brand-600 text-white' : 'text-gray-400 hover:text-gray-600')}
            >
              Puck
            </button>
          </div>
          <button
            onClick={openVisualEditor}
            disabled={editSessionLoading}
            title="Opens storefront-frontend's embedded visual editor in a new tab -- the real live components, true WYSIWYG"
            className="flex items-center gap-1.5 rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-xs font-medium text-gray-600 transition-colors hover:bg-gray-50 disabled:opacity-50"
          >
            <Wand2 size={12} /> {editSessionLoading ? 'Opening…' : 'Edit visually'}
          </button>
          <div className="flex items-center rounded-lg border border-gray-200 bg-white p-0.5">
            <button
              aria-label="Desktop"
              aria-pressed={breakpoint === 'desktop'}
              onClick={() => setBreakpoint('desktop')}
              className={clsx('rounded-md p-1.5 transition-colors', breakpoint === 'desktop' ? 'bg-brand-600 text-white' : 'text-gray-400 hover:text-gray-600')}
            >
              <Monitor size={14} />
            </button>
            <button
              aria-label="Mobile"
              aria-pressed={breakpoint === 'mobile'}
              onClick={() => setBreakpoint('mobile')}
              className={clsx('rounded-md p-1.5 transition-colors', breakpoint === 'mobile' ? 'bg-brand-600 text-white' : 'text-gray-400 hover:text-gray-600')}
            >
              <Smartphone size={14} />
            </button>
          </div>
          {slug === 'home' && (
            <div className="relative">
              <button
                onClick={() => setTemplatesOpen(o => !o)}
                className="flex items-center gap-1.5 rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-xs font-medium text-gray-600 transition-colors hover:bg-gray-50"
              >
                Template
              </button>
              {templatesOpen && (
                <div className="absolute right-0 z-10 mt-1 w-56 rounded-lg border border-gray-200 bg-white p-2 shadow-lg">
                  <p className="mb-1.5 px-1 text-[10px] font-medium uppercase tracking-wide text-gray-400">Starter template + color</p>
                  <div role="group" aria-label="Starter template" className="mb-2 flex gap-1 px-1">
                    {STARTER_TEMPLATES.map(t => (
                      <button
                        key={t.id}
                        type="button"
                        aria-pressed={selectedTemplateId === t.id}
                        onClick={() => setSelectedTemplateId(t.id)}
                        className={clsx(
                          'flex-1 rounded-md px-2 py-1 text-[11px] font-medium transition-colors',
                          selectedTemplateId === t.id ? 'bg-brand-600 text-white' : 'bg-gray-100 text-gray-600 hover:bg-gray-200',
                        )}
                      >
                        {t.label}
                      </button>
                    ))}
                  </div>
                  {THEME_PALETTES.map(palette => (
                    <button
                      key={palette.id}
                      onClick={() => applyTemplate(palette)}
                      className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs text-gray-600 hover:bg-gray-50"
                    >
                      <span
                        className="h-4 w-4 shrink-0 rounded-full border border-gray-200"
                        style={{ backgroundColor: palette.colors[500] }}
                      />
                      {palette.label}
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}
          {previewURL ? (
            <a
              href={previewURL}
              target="_blank"
              rel="noreferrer"
              title="Opens the real, saved, live-rendered page (not this canvas's rough layout preview) in a new tab"
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
          <div className="relative">
            <button
              onClick={() => setAddMenuOpen(o => !o)}
              className="flex items-center gap-1.5 rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-xs font-medium text-gray-600 transition-colors hover:bg-gray-50"
            >
              <Plus size={12} /> Add block
            </button>
            {addMenuOpen && (
              <div className="absolute right-0 z-10 mt-1 w-40 rounded-lg border border-gray-200 bg-white py-1 shadow-lg">
                {EDITABLE_BLOCK_TYPES.map(type => (
                  <button
                    key={type}
                    onClick={() => addBlock(type)}
                    className="block w-full px-3 py-1.5 text-left text-xs text-gray-600 hover:bg-gray-50"
                  >
                    {BLOCK_TYPE_LABELS[type]}
                  </button>
                ))}
              </div>
            )}
          </div>
          <button
            onClick={handleSave}
            disabled={saving || !dirty}
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
            <div
              id="theme-editor-canvas"
              data-testid="theme-canvas"
              className="relative mx-auto bg-white shadow-sm"
              style={{ width: displayedWidth, height: CANVAS_HEIGHT }}
              onClick={() => selectBlock(null)}
            >
              {customCss && <style>{scopeCssToCanvas(customCss, '#theme-editor-canvas')}</style>}
              {blocks.map(block => {
                const rect = rectFor(block)
                return (
                  <Rnd
                    key={block.id}
                    bounds="parent"
                    size={{ width: rect.width, height: rect.height }}
                    position={{ x: rect.x, y: rect.y }}
                    onDragStop={(_e, d) => updateRect(block.id, { ...rect, x: d.x, y: d.y })}
                    onResizeStop={(_e, _dir, ref, _delta, pos) => updateRect(block.id, {
                      x: pos.x, y: pos.y, width: parseInt(ref.style.width, 10), height: parseInt(ref.style.height, 10),
                    })}
                    onClick={(e: React.MouseEvent) => { e.stopPropagation(); selectBlock(block.id) }}
                    className={selectedBlockId === block.id ? 'ring-2 ring-brand-500' : 'ring-1 ring-gray-200'}
                  >
                    <BlockPreview block={block} />
                  </Rnd>
                )
              })}
            </div>
          )}
        </div>

        {cssPanelOpen && (
          <div className="w-80 shrink-0 overflow-y-auto border-l border-gray-200 bg-white">
            <CustomCSSPanel value={customCss} onChange={v => { setCustomCss(v); setDirty(true) }} />
          </div>
        )}
        {!cssPanelOpen && selectedBlock && (
          <div className="w-64 shrink-0 overflow-y-auto border-l border-gray-200 bg-white">
            <BlockPropsPanel
              token={token}
              block={selectedBlock}
              onChangeProps={props => updateBlock(selectedBlock.id, b => ({ ...b, props }))}
              onDelete={() => deleteBlock(selectedBlock.id)}
            />
          </div>
        )}
      </div>
    </div>
  )
}

import { useEffect, useRef, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { Rnd } from 'react-rnd'
import { ArrowLeft, Plus } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import { useThemePage } from '../theme/hooks/useThemePage'
import { themeEditorParamsFromPath, themeEditorPath } from '../theme/routing'
import { createBlock } from '../theme/blockDefaults'
import { BlockPreview } from '../theme/components/BlockPreview'
import { BlockPropsPanel } from '../theme/components/BlockPropsPanel'
import { PAGE_SLUGS, PAGE_SLUG_LABELS, EDITABLE_BLOCK_TYPES, BLOCK_TYPE_LABELS } from '../theme/types'
import type { Block, PageSlug } from '../theme/types'

const CANVAS_HEIGHT = 900

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
  const [selectedBlockId, setSelectedBlockId] = useState<string | null>(null)
  const [addMenuOpen, setAddMenuOpen] = useState(false)
  const [dirty, setDirty] = useState(false)
  const initializedFor = useRef<string | null>(null)

  useEffect(() => {
    const key = `${channelId}|${slug}`
    if (page && initializedFor.current !== key) {
      setBlocks(page.blocks)
      setCanvasWidth(page.canvas_width)
      setSelectedBlockId(null)
      setDirty(false)
      initializedFor.current = key
    }
  }, [page, channelId, slug])

  const selectedBlock = blocks.find(b => b.id === selectedBlockId) ?? null

  const updateBlock = (id: string, updater: (b: Block) => Block) => {
    setBlocks(prev => prev.map(b => (b.id === id ? updater(b) : b)))
    setDirty(true)
  }

  const addBlock = (type: (typeof EDITABLE_BLOCK_TYPES)[number]) => {
    const block = createBlock(type)
    setBlocks(prev => [...prev, block])
    setSelectedBlockId(block.id)
    setDirty(true)
    setAddMenuOpen(false)
  }

  const deleteBlock = (id: string) => {
    setBlocks(prev => prev.filter(b => b.id !== id))
    if (selectedBlockId === id) setSelectedBlockId(null)
    setDirty(true)
  }

  const handleSave = async () => {
    await save({ canvas_width: canvasWidth, blocks })
    setDirty(false)
  }

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
              data-testid="theme-canvas"
              className="relative mx-auto bg-white shadow-sm"
              style={{ width: canvasWidth, height: CANVAS_HEIGHT }}
              onClick={() => setSelectedBlockId(null)}
            >
              {blocks.map(block => (
                <Rnd
                  key={block.id}
                  bounds="parent"
                  size={{ width: block.layout.desktop.width, height: block.layout.desktop.height }}
                  position={{ x: block.layout.desktop.x, y: block.layout.desktop.y }}
                  onDragStop={(_e, d) => updateBlock(block.id, b => ({
                    ...b, layout: { ...b.layout, desktop: { ...b.layout.desktop, x: d.x, y: d.y } },
                  }))}
                  onResizeStop={(_e, _dir, ref, _delta, pos) => updateBlock(block.id, b => ({
                    ...b,
                    layout: {
                      ...b.layout,
                      desktop: { x: pos.x, y: pos.y, width: parseInt(ref.style.width, 10), height: parseInt(ref.style.height, 10) },
                    },
                  }))}
                  onClick={(e: React.MouseEvent) => { e.stopPropagation(); setSelectedBlockId(block.id) }}
                  className={selectedBlockId === block.id ? 'ring-2 ring-brand-500' : 'ring-1 ring-gray-200'}
                >
                  <BlockPreview block={block} />
                </Rnd>
              ))}
            </div>
          )}
        </div>

        {selectedBlock && (
          <div className="w-64 shrink-0 overflow-y-auto border-l border-gray-200 bg-white">
            <BlockPropsPanel
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

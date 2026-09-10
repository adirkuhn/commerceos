import { Trash2 } from 'lucide-react'
import { useCategories } from '../../pms/hooks/useCategories'
import { BLOCK_TYPE_LABELS } from '../types'
import type { Block } from '../types'

interface Props {
  token: string
  block: Block
  onChangeProps: (props: Record<string, unknown>) => void
  onDelete: () => void
}

function stringProp(props: Record<string, unknown>, key: string): string {
  const v = props[key]
  return typeof v === 'string' ? v : ''
}

function numberProp(props: Record<string, unknown>, key: string, fallback: number): number {
  const v = props[key]
  return typeof v === 'number' ? v : fallback
}

function idListProp(props: Record<string, unknown>, key: string): string[] {
  const v = props[key]
  return Array.isArray(v) ? v.filter((x): x is string => typeof x === 'string') : []
}

export function BlockPropsPanel({ token, block, onChangeProps, onDelete }: Props) {
  const set = (key: string, value: string) => onChangeProps({ ...block.props, [key]: value })
  const setNumber = (key: string, value: number) => onChangeProps({ ...block.props, [key]: value })
  const setIdList = (key: string, raw: string) => onChangeProps({
    ...block.props, [key]: raw.split(',').map(id => id.trim()).filter(id => id !== ''),
  })
  // Only fetched for the one block type that needs it -- useCategories
  // itself already no-ops without a token, so this is cheap for every
  // other block type too.
  const { categories } = useCategories(block.type === 'featured_products' ? token : '')

  return (
    <div className="flex h-full flex-col gap-4 p-4">
      <div className="flex items-center justify-between">
        <h3 className="text-xs font-semibold uppercase tracking-wider text-gray-500">
          {BLOCK_TYPE_LABELS[block.type]}
        </h3>
        <button
          onClick={onDelete}
          aria-label="Delete block"
          className="text-gray-400 hover:text-red-500 transition-colors"
        >
          <Trash2 size={14} />
        </button>
      </div>

      {block.type === 'hero' && (
        <>
          <Field label="Heading" value={stringProp(block.props, 'heading')} onChange={v => set('heading', v)} />
          <Field label="Subheading" value={stringProp(block.props, 'subheading')} onChange={v => set('subheading', v)} />
          <Field label="Image URL" value={stringProp(block.props, 'imageUrl')} onChange={v => set('imageUrl', v)} />
          <Field label="Button text" value={stringProp(block.props, 'ctaText')} onChange={v => set('ctaText', v)} />
          <Field label="Button link" value={stringProp(block.props, 'ctaUrl')} onChange={v => set('ctaUrl', v)} />
        </>
      )}

      {block.type === 'featured_products' && (
        <>
          <Field label="Title" value={stringProp(block.props, 'title')} onChange={v => set('title', v)} />
          <Field label="Subtitle" value={stringProp(block.props, 'subtitle')} onChange={v => set('subtitle', v)} />
          <label className="block text-xs">
            <span className="mb-1 block font-medium text-gray-500">Category (ignored if Product IDs below is set)</span>
            <select
              value={stringProp(block.props, 'categoryId')}
              onChange={e => set('categoryId', e.target.value)}
              className="w-full rounded-lg border border-gray-200 px-2.5 py-1.5 text-xs text-gray-700 outline-none focus:ring-1 focus:ring-brand-500"
            >
              <option value="">All categories</option>
              {categories.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
            </select>
          </label>
          <Field
            label="Product IDs (comma-separated, overrides Category)"
            value={idListProp(block.props, 'productIds').join(', ')}
            onChange={v => setIdList('productIds', v)}
          />
          <Field
            label="Exclude product IDs (comma-separated)"
            value={idListProp(block.props, 'excludeProductIds').join(', ')}
            onChange={v => setIdList('excludeProductIds', v)}
          />
          <label className="block text-xs">
            <span className="mb-1 block font-medium text-gray-500">Number of products</span>
            <input
              type="number"
              min={1}
              value={numberProp(block.props, 'limit', 8)}
              onChange={e => setNumber('limit', Number(e.target.value) || 1)}
              className="w-full rounded-lg border border-gray-200 px-2.5 py-1.5 text-xs text-gray-700 outline-none focus:ring-1 focus:ring-brand-500"
            />
          </label>
          <label className="block text-xs">
            <span className="mb-1 block font-medium text-gray-500">Items per row (desktop)</span>
            <input
              type="number"
              min={1}
              max={6}
              value={numberProp(block.props, 'itemsPerRowDesktop', 4)}
              onChange={e => setNumber('itemsPerRowDesktop', Number(e.target.value) || 1)}
              className="w-full rounded-lg border border-gray-200 px-2.5 py-1.5 text-xs text-gray-700 outline-none focus:ring-1 focus:ring-brand-500"
            />
          </label>
          <label className="block text-xs">
            <span className="mb-1 block font-medium text-gray-500">Items per row (mobile)</span>
            <input
              type="number"
              min={1}
              max={3}
              value={numberProp(block.props, 'itemsPerRowMobile', 2)}
              onChange={e => setNumber('itemsPerRowMobile', Number(e.target.value) || 1)}
              className="w-full rounded-lg border border-gray-200 px-2.5 py-1.5 text-xs text-gray-700 outline-none focus:ring-1 focus:ring-brand-500"
            />
          </label>
        </>
      )}

      {block.type === 'rich_text' && (
        <TextAreaField label="Content" value={stringProp(block.props, 'content')} onChange={v => set('content', v)} />
      )}

      {block.type === 'custom_html' && (
        <TextAreaField label="HTML" value={stringProp(block.props, 'html')} onChange={v => set('html', v)} monospace />
      )}
    </div>
  )
}

function Field({ label, value, onChange }: { label: string; value: string; onChange: (v: string) => void }) {
  return (
    <label className="block text-xs">
      <span className="mb-1 block font-medium text-gray-500">{label}</span>
      <input
        type="text"
        value={value}
        onChange={e => onChange(e.target.value)}
        className="w-full rounded-lg border border-gray-200 px-2.5 py-1.5 text-xs text-gray-700 outline-none focus:ring-1 focus:ring-brand-500"
      />
    </label>
  )
}

function TextAreaField({ label, value, onChange, monospace }: { label: string; value: string; onChange: (v: string) => void; monospace?: boolean }) {
  return (
    <label className="block text-xs">
      <span className="mb-1 block font-medium text-gray-500">{label}</span>
      <textarea
        value={value}
        onChange={e => onChange(e.target.value)}
        rows={6}
        className={`w-full rounded-lg border border-gray-200 px-2.5 py-1.5 text-xs text-gray-700 outline-none focus:ring-1 focus:ring-brand-500 ${monospace ? 'font-mono' : ''}`}
      />
    </label>
  )
}

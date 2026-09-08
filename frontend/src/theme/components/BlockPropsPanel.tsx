import { Trash2 } from 'lucide-react'
import { BLOCK_TYPE_LABELS } from '../types'
import type { Block } from '../types'

interface Props {
  block: Block
  onChangeProps: (props: Record<string, unknown>) => void
  onDelete: () => void
}

function stringProp(props: Record<string, unknown>, key: string): string {
  const v = props[key]
  return typeof v === 'string' ? v : ''
}

export function BlockPropsPanel({ block, onChangeProps, onDelete }: Props) {
  const set = (key: string, value: string) => onChangeProps({ ...block.props, [key]: value })

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

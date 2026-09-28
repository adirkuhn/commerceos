import type { Editor } from 'grapesjs'
import type { DocumentSummary } from '../core/document'
import { DEVICES, type DeviceId } from './useGrapesEditor'
import type { DocRef, Status } from './useDraft'

const KIND_LABELS: Record<DocumentSummary['kind'], string> = { template: 'Templates', section_group: 'Header & footer', page: 'Pages' }

const labelFor = (d: DocumentSummary) =>
  d.title || d.key.split('-').map(w => w[0].toUpperCase() + w.slice(1)).join(' ')

interface Props {
  themeName: string
  documents: DocumentSummary[]
  doc: DocRef
  onDoc: (d: DocRef) => void
  editor?: Editor
  device: DeviceId
  onDevice: (d: DeviceId) => void
  dirty: boolean
  status: Status
  onSave: () => void
  onPreview: () => void
  onPublish: () => void
}

const btn = 'px-3 py-1.5 rounded-md text-sm font-medium disabled:opacity-40'

export function Toolbar(p: Props) {
  const busy = p.status.kind === 'busy'
  return (
    <header className="flex items-center gap-3 border-b border-neutral-200 bg-white px-4 py-2">
      <span className="font-semibold text-neutral-900 truncate max-w-48" title={p.themeName}>{p.themeName}</span>
      <label className="sr-only" htmlFor="doc-select">Page</label>
      <select
        id="doc-select"
        className="rounded-md border border-neutral-300 px-2 py-1.5 text-sm"
        value={`${p.doc.kind}:${p.doc.key}`}
        onChange={e => {
          const [kind, key] = e.target.value.split(':')
          p.onDoc({ kind: kind as DocRef['kind'], key })
        }}
      >
        {(Object.keys(KIND_LABELS) as DocumentSummary['kind'][]).map(kind => (
          <optgroup key={kind} label={KIND_LABELS[kind]}>
            {p.documents.filter(d => d.kind === kind).map(d => (
              <option key={d.key} value={`${d.kind}:${d.key}`}>{labelFor(d)}</option>
            ))}
          </optgroup>
        ))}
      </select>

      <div className="flex rounded-md border border-neutral-300" role="group" aria-label="Preview size">
        {DEVICES.map(d => (
          <button key={d.id} type="button" aria-pressed={p.device === d.id}
            className={`${btn} ${p.device === d.id ? 'bg-neutral-900 text-white' : 'text-neutral-700'}`}
            onClick={() => p.onDevice(d.id)}>{d.name}</button>
        ))}
      </div>

      <button type="button" className={btn} onClick={() => p.editor?.UndoManager.undo()} disabled={!p.editor} aria-label="Undo">↶</button>
      <button type="button" className={btn} onClick={() => p.editor?.UndoManager.redo()} disabled={!p.editor} aria-label="Redo">↷</button>

      <span className="ml-auto text-sm text-neutral-600" role="status" aria-live="polite">
        {p.status.kind === 'busy' && p.status.label}
        {p.status.kind === 'ok' && p.status.message}
        {p.status.kind === 'error' && <span className="text-red-700">{p.status.message}</span>}
        {p.status.kind === 'idle' && (p.dirty ? 'Unsaved changes' : '')}
      </span>
      <button type="button" className={`${btn} border border-neutral-300`} onClick={p.onSave} disabled={busy || !p.dirty}>Save</button>
      <button type="button" className={`${btn} border border-neutral-300`} onClick={p.onPreview} disabled={busy}>Preview</button>
      <button type="button" className={`${btn} bg-blue-700 text-white`} onClick={p.onPublish} disabled={busy}>Publish</button>
    </header>
  )
}

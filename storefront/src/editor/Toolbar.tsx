import type { Editor } from 'grapesjs'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import type { DocumentSummary } from '../core/document'
import { DEVICES, type DeviceId } from './useGrapesEditor'
import type { DocRef, Status } from './useDraft'
import { useEditorLabels } from './i18n/useEditorLabels'

const KINDS: DocumentSummary['kind'][] = ['template', 'section_group', 'page']

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
  /** Extra controls after the page picker (the texts' language). */
  children?: ReactNode
}

const btn = 'px-3 py-1.5 rounded-md text-sm font-medium disabled:opacity-40'

export function Toolbar(p: Props) {
  const busy = p.status.kind === 'busy'
  const { t } = useTranslation('editor')
  const labels = useEditorLabels()
  const kindLabels: Record<DocumentSummary['kind'], string> = { template: t('toolbar.templates'), section_group: t('toolbar.sectionGroups'), page: t('toolbar.pages') }
  const deviceLabels: Record<DeviceId, string> = { desktop: t('toolbar.devices.desktop'), tablet: t('toolbar.devices.tablet'), mobile: t('toolbar.devices.mobile') }
  return (
    <header className="flex items-center gap-3 border-b border-neutral-200 bg-white px-4 py-2">
      <span className="font-semibold text-neutral-900 truncate max-w-48" title={p.themeName}>{p.themeName}</span>
      <label className="sr-only" htmlFor="doc-select">{t('toolbar.page')}</label>
      <select
        id="doc-select"
        className="rounded-md border border-neutral-300 px-2 py-1.5 text-sm"
        value={`${p.doc.kind}:${p.doc.key}`}
        onChange={e => {
          const [kind, key] = e.target.value.split(':')
          p.onDoc({ kind: kind as DocRef['kind'], key })
        }}
      >
        {KINDS.map(kind => (
          <optgroup key={kind} label={kindLabels[kind]}>
            {p.documents.filter(d => d.kind === kind).map(d => (
              <option key={d.key} value={`${d.kind}:${d.key}`}>{labels.document(d)}</option>
            ))}
          </optgroup>
        ))}
      </select>
      {p.children}

      <div className="flex rounded-md border border-neutral-300" role="group" aria-label={t('toolbar.previewSize')}>
        {DEVICES.map(d => (
          <button key={d.id} type="button" aria-pressed={p.device === d.id}
            className={`${btn} ${p.device === d.id ? 'bg-neutral-900 text-white' : 'text-neutral-700'}`}
            onClick={() => p.onDevice(d.id)}>{deviceLabels[d.id]}</button>
        ))}
      </div>

      <button type="button" className={btn} onClick={() => p.editor?.UndoManager.undo()} disabled={!p.editor} aria-label={t('toolbar.undo')}>↶</button>
      <button type="button" className={btn} onClick={() => p.editor?.UndoManager.redo()} disabled={!p.editor} aria-label={t('toolbar.redo')}>↷</button>

      <span className="ml-auto text-sm text-neutral-600" role="status" aria-live="polite">
        {p.status.kind === 'busy' && p.status.label}
        {p.status.kind === 'ok' && p.status.message}
        {p.status.kind === 'error' && <span className="text-red-700">{p.status.message}</span>}
        {p.status.kind === 'idle' && (p.dirty ? t('toolbar.unsavedChanges') : '')}
      </span>
      <button type="button" className={`${btn} border border-neutral-300`} onClick={p.onSave} disabled={busy || !p.dirty}>{t('toolbar.save')}</button>
      <button type="button" className={`${btn} border border-neutral-300`} onClick={p.onPreview} disabled={busy}>{t('toolbar.preview')}</button>
      <button type="button" className={`${btn} bg-blue-700 text-white`} onClick={p.onPublish} disabled={busy}>{t('toolbar.publish')}</button>
    </header>
  )
}

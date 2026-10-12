import type { Editor } from 'grapesjs'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import type { DocumentSummary } from '../core/document'
import { DEVICES, type DeviceId } from './useGrapesEditor'
import type { DocRef, Status } from './useDraft'
import { useEditorLabels } from './i18n/useEditorLabels'
import { DesktopIcon, EyeIcon, MobileIcon, RedoIcon, TabletIcon, UndoIcon } from './icons'

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

const btn = 'px-3 py-1.5 rounded-ed text-sm font-medium transition-colors disabled:opacity-40'
const iconBtn = 'p-1.5 rounded-ed text-ed-ink-3 transition-colors hover:bg-ed-hover hover:text-ed-ink disabled:opacity-40'
const DEVICE_ICONS: Record<DeviceId, typeof DesktopIcon> = { desktop: DesktopIcon, tablet: TabletIcon, mobile: MobileIcon }

export function Toolbar(p: Props) {
  const busy = p.status.kind === 'busy'
  const { t } = useTranslation('editor')
  const labels = useEditorLabels()
  const kindLabels: Record<DocumentSummary['kind'], string> = { template: t('toolbar.templates'), section_group: t('toolbar.sectionGroups'), page: t('toolbar.pages') }
  const deviceLabels: Record<DeviceId, string> = { desktop: t('toolbar.devices.desktop'), tablet: t('toolbar.devices.tablet'), mobile: t('toolbar.devices.mobile') }
  return (
    <header className="flex items-center gap-3 border-b border-ed-line bg-ed-surface px-4 py-2">
      <span className="max-w-48 truncate font-semibold text-ed-ink" title={p.themeName}>{p.themeName}</span>
      <label className="sr-only" htmlFor="doc-select">{t('toolbar.page')}</label>
      <select
        id="doc-select"
        className="rounded-ed border border-ed-line bg-ed-sunken px-2 py-1.5 text-sm text-ed-ink focus:outline-none focus:ring-2 focus:ring-ed-accent/60"
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

      <div className="flex items-center gap-0.5 rounded-ed bg-ed-fill p-0.5" role="group" aria-label={t('toolbar.previewSize')}>
        {DEVICES.map(d => {
          const Icon = DEVICE_ICONS[d.id]
          const active = p.device === d.id
          return (
            <button key={d.id} type="button" aria-pressed={active} aria-label={deviceLabels[d.id]} title={deviceLabels[d.id]}
              className={`rounded-ed p-1.5 transition-colors ${active ? 'bg-ed-raised text-ed-ink shadow-sm' : 'text-ed-ink-3 hover:text-ed-ink'}`}
              onClick={() => p.onDevice(d.id)}>
              <Icon size={16} />
            </button>
          )
        })}
      </div>

      <div className="flex items-center gap-0.5">
        <button type="button" className={iconBtn} onClick={() => p.editor?.UndoManager.undo()} disabled={!p.editor} aria-label={t('toolbar.undo')} title={t('toolbar.undo')}><UndoIcon /></button>
        <button type="button" className={iconBtn} onClick={() => p.editor?.UndoManager.redo()} disabled={!p.editor} aria-label={t('toolbar.redo')} title={t('toolbar.redo')}><RedoIcon /></button>
      </div>

      <span className="ml-auto text-sm text-ed-ink-3" role="status" aria-live="polite">
        {p.status.kind === 'busy' && p.status.label}
        {p.status.kind === 'ok' && <span className="text-ed-success">{p.status.message}</span>}
        {p.status.kind === 'error' && <span className="text-ed-danger">{p.status.message}</span>}
        {p.status.kind === 'idle' && (p.dirty ? t('toolbar.unsavedChanges') : '')}
      </span>
      <button type="button" className={`${btn} bg-ed-fill text-ed-ink-2 hover:bg-ed-hover`} onClick={p.onSave} disabled={busy || !p.dirty}>{t('toolbar.save')}</button>
      <button type="button" className={`${btn} inline-flex items-center gap-1.5 text-ed-ink-2 hover:bg-ed-hover`} onClick={p.onPreview} disabled={busy}>
        <EyeIcon size={15} />{t('toolbar.preview')}
      </button>
      <button type="button" className={`${btn} bg-ed-accent-solid font-semibold text-white hover:bg-ed-accent-solid-hover`} onClick={p.onPublish} disabled={busy}>{t('toolbar.publish')}</button>
    </header>
  )
}

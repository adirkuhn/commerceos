import { useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, Code2, Monitor, Redo2, RotateCcw, Smartphone, Undo2 } from 'lucide-react'
import clsx from 'clsx'
import { useConfirm } from '../../components/common/useConfirm'
import { useAuth } from '../../auth/AuthContext'
import { apiErrorText } from '../../api/apiErrorText'
import * as api from '../api'
import type { MessageTemplate, MessageTrigger, TemplatePreview, TriggerSubject } from '../types'
import { triggerLabels } from '../triggerLabels'
import { emailBlocks, type BlockTexts } from './emailBlocks'
import { toEditorBlocks } from './editorBlocks'
import { previewTriggerFor } from './designHelpers'
import { useEmailEditor } from './useEmailEditor'
import { VariablesPanel } from './VariablesPanel'
import { EditorSidePanel, type SidePanelTab } from './EditorSidePanel'
import { PreviewDialog } from './PreviewDialog'

interface Props {
  template: MessageTemplate
  onSave: (template: MessageTemplate, changes: { subject: string; body: string; design: Record<string, unknown> }) => Promise<void>
  // Set for a template made from a default: puts it back to that default.
  onReset?: (template: MessageTemplate) => Promise<void>
  onBack: () => void
}

// EmailDesignEditor is the visual email editor (GrapesJS + newsletter
// preset) for one template: blocks and styles in the canvas, the trigger's
// variables on the side, a server-rendered preview, and save -- which
// stores the inline-styled HTML that is sent and the design to reopen.
export function EmailDesignEditor({ template, onSave, onReset, onBack }: Props) {
  const { confirm, confirmDialog } = useConfirm()
  const { token } = useAuth()
  const { t, i18n } = useTranslation(['messages', 'common'])
  const canvasRef = useRef<HTMLDivElement>(null)
  const blocksRef = useRef<HTMLDivElement>(null)
  const stylesRef = useRef<HTMLDivElement>(null)
  const layersRef = useRef<HTMLDivElement>(null)
  const [tab, setTab] = useState<SidePanelTab>('blocks')
  const [triggers, setTriggers] = useState<MessageTrigger[]>([])
  const [subject, setSubject] = useState(template.subject)
  const [previewAs, setPreviewAs] = useState<TriggerSubject>(previewTriggerFor(template.category))
  const [preview, setPreview] = useState<TemplatePreview | null>(null)
  const [busy, setBusy] = useState<'' | 'saving' | 'previewing' | 'resetting'>('')
  const [status, setStatus] = useState<{ kind: 'ok' | 'error'; text: string } | null>(null)

  const blockTexts = t('design.blockTexts', { returnObjects: true }) as BlockTexts
  const blocks = useMemo(() => toEditorBlocks(emailBlocks(blockTexts), key => t(key)), [blockTexts, t])
  const editor = useEmailEditor(canvasRef, {
    design: template.design, html: template.body, blocks, blockCategory: 'CommerceOS', locale: i18n.language,
    panels: { blocks: blocksRef, styles: stylesRef, layers: layersRef },
    sectorLabels: {
      text: t('design.sectors.text'), spacing: t('design.sectors.spacing'), size: t('design.sectors.size'),
      background: t('design.sectors.background'), border: t('design.sectors.border'),
    },
    onSelect: () => setTab('style'),
    uploadImage: file => api.uploadMessageTemplateImage(token ?? '', file),
    onUploadError: err => setStatus({ kind: 'error', text: apiErrorText(err) ?? t('design.uploadFailed') }),
  })

  useEffect(() => {
    api.listMessageTriggers(token ?? '').then(setTriggers, err => setStatus({ kind: 'error', text: apiErrorText(err) ?? t('design.loadFailed') }))
  }, [token, t])

  const labels = triggerLabels(t)
  const trigger = triggers.find(tr => tr.subject === previewAs)

  async function save() {
    setBusy('saving')
    setStatus(null)
    try {
      const { html, design } = editor.content()
      await onSave(template, { subject, body: html, design })
      setStatus({ kind: 'ok', text: t('common:status.saved') })
    } catch (err) {
      setStatus({ kind: 'error', text: apiErrorText(err) ?? t('design.saveFailed') })
    } finally {
      setBusy('')
    }
  }

  async function resetToDefault() {
    if (!onReset) return
    if (!(await confirm({ title: t('design.resetTitle'), message: t('design.resetMessage'), confirmLabel: t('design.resetConfirm'), destructive: true }))) return
    setBusy('resetting')
    setStatus(null)
    try {
      await onReset(template)
    } catch (err) {
      setStatus({ kind: 'error', text: apiErrorText(err) ?? t('design.resetFailed') })
      setBusy('')
    }
  }

  async function showPreview() {
    setBusy('previewing')
    setStatus(null)
    try {
      const { html } = editor.content()
      setPreview(await api.previewMessageTemplate(token ?? '', {
        trigger_subject: previewAs, subject, body: html, text_body: template.text_body,
      }))
    } catch (err) {
      setStatus({ kind: 'error', text: apiErrorText(err) ?? t('design.previewFailed') })
    } finally {
      setBusy('')
    }
  }

  const iconButton = 'p-1.5 rounded-lg text-ink-3 hover:bg-hover hover:text-ink disabled:opacity-40'
  const devices = [
    { id: 'desktop' as const, icon: Monitor, label: t('design.desktop') },
    { id: 'mobile' as const, icon: Smartphone, label: t('design.mobile') },
  ]

  return (
    <div className="flex flex-col h-full">
      <header className="shrink-0 border-b border-line bg-surface">
        <div className="flex items-center gap-3 px-4 pt-3 pb-2">
          <button onClick={onBack} aria-label={t('common:actions.back')} className={iconButton}>
            <ArrowLeft size={16} />
          </button>
          <h2 className="text-base font-semibold text-ink truncate">{template.name}</h2>
          <label className="flex items-center gap-2 flex-1 min-w-[220px] ml-4 text-xs font-medium text-ink-3">
            {t('design.previewSubject')}
            <input value={subject} onChange={e => setSubject(e.target.value)}
              className="flex-1 px-3 py-1.5 rounded-lg bg-sunken border border-line text-sm text-ink outline-none focus:ring-2 focus:ring-focus/40" />
          </label>
        </div>
        <div className="flex flex-wrap items-center gap-2 px-4 pb-2.5">
          <div className="flex items-center gap-0.5">
            <button onClick={editor.undo} disabled={!editor.ready} aria-label={t('design.undo')} title={t('design.undo')} className={iconButton}><Undo2 size={16} /></button>
            <button onClick={editor.redo} disabled={!editor.ready} aria-label={t('design.redo')} title={t('design.redo')} className={iconButton}><Redo2 size={16} /></button>
          </div>
          <div className="flex items-center gap-0.5 p-0.5 rounded-lg bg-fill">
            {devices.map(({ id, icon: Icon, label }) => (
              <button key={id} onClick={() => editor.setDevice(id)} disabled={!editor.ready} aria-pressed={editor.device === id}
                aria-label={label} title={label}
                className={clsx('p-1.5 rounded-md transition-colors disabled:opacity-40', editor.device === id ? 'bg-raised text-ink shadow-sm' : 'text-ink-3 hover:text-ink')}>
                <Icon size={16} />
              </button>
            ))}
          </div>
          <button onClick={editor.showCode} disabled={!editor.ready} aria-label={t('design.viewCode')} title={t('design.viewCode')} className={iconButton}><Code2 size={16} /></button>
          <div className="flex-1" />
          {status && <span role={status.kind === 'error' ? 'alert' : 'status'} className={status.kind === 'error' ? 'text-xs text-danger' : 'text-xs text-success'}>{status.text}</span>}
          <label className="flex items-center gap-2 text-xs text-ink-3">
            {t('design.previewAs')}
            <select value={previewAs} onChange={e => setPreviewAs(e.target.value as TriggerSubject)}
              className="max-w-[14rem] px-2 py-1.5 rounded-lg bg-sunken border border-line text-xs text-ink-2">
              {triggers.map(tr => <option key={tr.subject} value={tr.subject}>{labels[tr.subject] ?? tr.subject}</option>)}
            </select>
          </label>
          {onReset && (
            <button onClick={resetToDefault} disabled={busy !== ''} title={t('design.reset')}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-ink-3 hover:bg-hover disabled:opacity-50">
              <RotateCcw size={14} /> {t('design.reset')}
            </button>
          )}
          <button onClick={showPreview} disabled={busy !== '' || !editor.ready}
            className="px-3 py-1.5 rounded-lg bg-fill text-xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50">
            {busy === 'previewing' ? t('common:status.loading') : t('design.preview')}
          </button>
          <button onClick={save} disabled={busy !== '' || !editor.ready}
            className="px-4 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50">
            {busy === 'saving' ? t('common:actions.saving') : t('common:actions.save')}
          </button>
        </div>
      </header>
      <div className="flex flex-1 min-h-0">
        <div ref={canvasRef} className="cos-email-editor flex-1 min-w-0" data-testid="email-canvas" />
        <EditorSidePanel
          tab={tab}
          onTab={setTab}
          refs={{ blocks: blocksRef, styles: stylesRef, layers: layersRef }}
          variables={trigger
            ? <VariablesPanel variables={trigger.variables} onInsertText={editor.insertText} onInsertHTML={editor.insertHTML} />
            : <p className="p-3 text-xs text-ink-4">{t('common:status.loading')}</p>}
        />
      </div>
      {preview && <PreviewDialog preview={preview} onClose={() => setPreview(null)} />}
      {confirmDialog}
    </div>
  )
}

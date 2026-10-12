import { useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, RotateCcw } from 'lucide-react'
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

  return (
    <div className="flex flex-col h-full">
      <div className="flex flex-wrap items-center gap-3 px-4 py-2.5 border-b border-line bg-sunken shrink-0">
        <button onClick={onBack} aria-label={t('common:actions.back')} className="p-1 rounded hover:bg-hover">
          <ArrowLeft size={16} className="text-ink-3" />
        </button>
        <h2 className="text-sm font-semibold text-ink">{template.name}</h2>
        <label className="flex items-center gap-2 text-xs text-ink-3 flex-1 min-w-[200px]">
          {t('design.previewSubject')}
          <input value={subject} onChange={e => setSubject(e.target.value)}
            className="flex-1 px-2 py-1 rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus" />
        </label>
        <label className="flex items-center gap-2 text-xs text-ink-3">
          {t('design.previewAs')}
          <select value={previewAs} onChange={e => setPreviewAs(e.target.value as TriggerSubject)}
            className="px-2 py-1 rounded-lg bg-surface border border-line text-ink-2">
            {triggers.map(tr => <option key={tr.subject} value={tr.subject}>{labels[tr.subject] ?? tr.subject}</option>)}
          </select>
        </label>
        {onReset && (
          <button onClick={resetToDefault} disabled={busy !== ''} title={t('design.reset')}
            className="flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs font-medium text-ink-3 hover:bg-hover disabled:opacity-50">
            <RotateCcw size={12} /> {t('design.reset')}
          </button>
        )}
        <button onClick={showPreview} disabled={busy !== '' || !editor.ready}
          className="px-3 py-1.5 rounded-lg bg-fill text-xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50">
          {busy === 'previewing' ? t('common:status.loading') : t('design.preview')}
        </button>
        <button onClick={save} disabled={busy !== '' || !editor.ready}
          className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50">
          {busy === 'saving' ? t('common:actions.saving') : t('common:actions.save')}
        </button>
        {status && <span role={status.kind === 'error' ? 'alert' : 'status'} className={status.kind === 'error' ? 'text-xs text-danger' : 'text-xs text-success'}>{status.text}</span>}
      </div>
      <div className="flex flex-1 min-h-0">
        <div ref={canvasRef} className="flex-1 min-w-0" data-testid="email-canvas" />
        <aside className="w-64 shrink-0 border-l border-line bg-raised overflow-y-auto">
          {trigger
            ? <VariablesPanel variables={trigger.variables} onInsertText={editor.insertText} onInsertHTML={editor.insertHTML} />
            : <p className="p-3 text-xs text-ink-4">{t('common:status.loading')}</p>}
        </aside>
      </div>
      {preview && <PreviewDialog preview={preview} onClose={() => setPreview(null)} />}
      {confirmDialog}
    </div>
  )
}

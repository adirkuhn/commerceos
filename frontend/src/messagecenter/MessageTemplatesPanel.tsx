import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Search, Plus, Pencil, Trash2, X, Mail, MessageSquare, Phone, Palette } from 'lucide-react'
import { useLocation, useNavigate } from 'react-router-dom'
import { subPathSegments } from '../routes'
import { useAuth } from '../auth/AuthContext'
import { TEMPLATES_PATH } from './messageCenterPaths'
import { useRoutedEditor, NEW_RECORD } from '../routing/useRoutedEditor'
import { useInitOnce } from '../routing/useInitOnce'
import { useMessageTemplates } from './hooks/useMessageTemplates'
import type { MessageTemplate, MessageChannel, MessageCategory, MessageTemplateDefault } from './types'
import { Dialog } from '../components/common/Dialog'
import { useConfirm } from '../components/common/useConfirm'
import { apiErrorText } from '../api/apiErrorText'
import { EmailDesignEditor } from './design/EmailDesignEditor'
import { StartFromSelect } from './design/StartFromSelect'
import { DESIGN_VIEW, templateDesignPath } from './messageCenterPaths'
import { plainTextExcerpt } from './design/designHelpers'


const CHANNEL_ICONS: Record<MessageChannel, typeof Mail> = {
  email: Mail,
  sms: Phone,
  whatsapp: MessageSquare,
}

const ALL_CHANNELS: MessageChannel[] = ['email', 'sms', 'whatsapp']
const ALL_CATEGORIES: MessageCategory[] = ['purchase', 'order_status', 'stock_info', 'tracking', 'custom']

interface FormState {
  name: string
  category: MessageCategory
  channels: MessageChannel[]
  subject: string
  body: string
  textBody: string
  enabled: boolean
  // The default a new template starts from ('' = blank).
  defaultKey: string
}

function emptyForm(): FormState {
  return { name: '', category: 'purchase', channels: [], subject: '', body: '', textBody: '', enabled: true, defaultKey: '' }
}

function toForm(tpl: MessageTemplate): FormState {
  return { name: tpl.name, category: tpl.category, channels: tpl.channels, subject: tpl.subject, body: tpl.body, textBody: tpl.text_body ?? '', enabled: tpl.enabled, defaultKey: tpl.default_key ?? '' }
}

export default function MessageTemplatesPanel() {
  const { confirm, confirmDialog } = useConfirm()
  const { token } = useAuth()
  const { t } = useTranslation(['messages', 'common'])
  const CATEGORY_LABELS: Record<MessageCategory, string> = {
    purchase: t('category.purchase'), order_status: t('category.order_status'), stock_info: t('category.stock_info'),
    tracking: t('category.tracking'), custom: t('category.custom'),
  }
  const CHANNEL_LABELS: Record<MessageChannel, string> = { email: t('channel.email'), sms: t('channel.sms'), whatsapp: t('channel.whatsapp') }
  const { templates, loading, error, create, update, reset, remove } = useMessageTemplates(token ?? '')

  const location = useLocation()
  const navigate = useNavigate()
  const [search, setSearch] = useState('')
  // /message-center/templates/<id>/design opens the visual email editor.
  const [designId, view] = subPathSegments(location.pathname, TEMPLATES_PATH)
  const designing = view === DESIGN_VIEW ? templates.find(tpl => tpl.id === designId) ?? null : null
  // Bumped by "reset to default" so the editor reloads the restored body.
  const [resets, setResets] = useState(0)
  // Editor open state is the URL (/message-center/templates/<new|id>).
  const editor = useRoutedEditor(TEMPLATES_PATH)
  const creating = editor.editingId === NEW_RECORD
  const editingTemplate = editor.editingId && !creating ? templates.find(tpl => tpl.id === editor.editingId) ?? null : null
  const [form, setForm] = useState<FormState>(emptyForm())
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')
  const [actionError, setActionError] = useState('')

  const q = search.toLowerCase()
  const filtered = templates.filter(tpl => tpl.name.toLowerCase().includes(q) || tpl.body.toLowerCase().includes(q))

  const formReady = useInitOnce(editor.editingId, creating || !!editingTemplate, () => {
    setForm(editingTemplate ? toForm(editingTemplate) : emptyForm())
    setFormError('')
  })
  const openCreate = editor.openNew
  const openEdit = (tpl: MessageTemplate) => editor.openEdit(tpl.id)
  const closeForm = editor.close

  function toggleChannel(ch: MessageChannel) {
    setForm(f => ({
      ...f,
      channels: f.channels.includes(ch) ? f.channels.filter(c => c !== ch) : [...f.channels, ch],
    }))
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setSaving(true)
    setFormError('')
    try {
      const content = { name: form.name, category: form.category, channels: form.channels, subject: form.subject, body: form.body, text_body: form.textBody }
      if (editingTemplate) {
        await update(editingTemplate.id, { ...content, enabled: form.enabled })
      } else {
        await create({ ...content, ...(form.defaultKey ? { default_key: form.defaultKey } : {}) })
      }
      closeForm()
    } catch (err) {
      setFormError(apiErrorText(err) ?? t('common:status.error'))
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete(tpl: MessageTemplate) {
    if (!(await confirm({ title: t('templates.deleteTitle', { name: tpl.name }), message: t('templates.deleteMessage'), confirmLabel: t('templates.deleteConfirm'), destructive: true }))) return
    setActionError('')
    try {
      await remove(tpl.id)
    } catch (err) {
      setActionError(apiErrorText(err) ?? t('common:status.error'))
    }
  }

  async function saveDesign(tpl: MessageTemplate, changes: { subject: string; body: string; design: Record<string, unknown> }) {
    await update(tpl.id, {
      name: tpl.name, category: tpl.category, channels: tpl.channels, text_body: tpl.text_body, enabled: tpl.enabled, ...changes,
    })
  }

  function startFrom(d: MessageTemplateDefault | null) {
    setForm(f => d
      ? { ...f, name: d.name, category: d.category, channels: d.channels, subject: d.subject, body: d.body, textBody: d.text_body, defaultKey: d.key }
      : { ...emptyForm(), enabled: f.enabled })
  }

  async function resetDesign(tpl: MessageTemplate) {
    await reset(tpl.id)
    setResets(n => n + 1)
  }

  if (view === DESIGN_VIEW) {
    if (designing) {
      return (
        <EmailDesignEditor key={`${designing.id}:${resets}`} template={designing} onSave={saveDesign}
          onReset={designing.default_key ? resetDesign : undefined} onBack={() => navigate(TEMPLATES_PATH)} />
      )
    }
    return <p className="p-6 text-xs text-ink-4">{loading ? t('common:status.loading') : t('design.notFound')}</p>
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <div className="relative flex-1 max-w-sm">
          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-4" />
          <input
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder={t('templates.search')}
            className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg bg-raised border-line border text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
        <button
          onClick={openCreate}
          className="ml-auto flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-accent-solid text-white hover:bg-accent-solid-hover transition-colors"
        >
          <Plus size={13} /> {t('templates.new')}
        </button>
      </div>

      <div className="flex items-center gap-6 px-5 py-2.5 border-b border-line bg-surface shrink-0">
        <span className="text-xs text-ink-4">{t('templates.count', { count: filtered.length })}</span>
        {(error || actionError) && <span className="text-xs text-danger ml-auto">{error || actionError}</span>}
      </div>

      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('templates.colName')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('templates.colCategory')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('templates.colChannels')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('templates.colBody')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('templates.colStatus')}</th>
              <th className="px-3 py-2.5 w-24" />
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {loading && (
              <tr><td colSpan={6} className="px-4 py-6 text-center text-ink-4">{t('common:status.loading')}</td></tr>
            )}
            {!loading && filtered.length === 0 && (
              <tr><td colSpan={6} className="px-4 py-6 text-center text-ink-4">{t('templates.empty')}</td></tr>
            )}
            {filtered.map(tpl => (
              <tr key={tpl.id} className="group hover:bg-hover/60 transition-colors">
                <td className="px-3 py-2.5 font-medium text-ink">{tpl.name}</td>
                <td className="px-3 py-2.5 text-ink-3">{CATEGORY_LABELS[tpl.category]}</td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1.5">
                    {tpl.channels.map(ch => {
                      const Icon = CHANNEL_ICONS[ch]
                      return <span key={ch} title={CHANNEL_LABELS[ch]}><Icon size={12} className="text-ink-3" /></span>
                    })}
                  </div>
                </td>
                <td className="px-3 py-2.5 max-w-[320px] truncate text-ink-3">{plainTextExcerpt(tpl.body || tpl.text_body || '')}</td>
                <td className="px-3 py-2.5">
                  <span className={tpl.enabled ? 'text-success' : 'text-ink-4'}>
                    {tpl.enabled ? t('common:status.enabled') : t('common:status.disabled')}
                  </span>
                </td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity">
                    {tpl.channels.includes('email') && (
                      <button aria-label={t('design.designEmail')}
                        onClick={() => navigate(templateDesignPath(tpl.id))}
                        className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
                        title={t('design.designEmail')}
                      >
                        <Palette size={12} className="text-ink-3" />
                      </button>
                    )}
                    <button aria-label={t('common:actions.edit')}
                      onClick={() => openEdit(tpl)}
                      className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
                      title={t('common:actions.edit')}
                    >
                      <Pencil size={12} className="text-ink-3" />
                    </button>
                    <button aria-label={t('common:actions.delete')}
                      onClick={() => handleDelete(tpl)}
                      className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
                      title={t('common:actions.delete')}
                    >
                      <Trash2 size={12} className="text-ink-3 hover:text-danger" />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {formReady && (
        <Dialog onClose={closeForm} className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={closeForm} />
          <div className={`relative w-full max-w-md bg-raised rounded-xl shadow-2xl border border-line p-5`}>
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-ink">
                {editingTemplate ? t('templates.editTitle') : t('templates.newTitle')}
              </h3>
              <button onClick={closeForm} aria-label={t('common:actions.close')}>
                <X size={14} className="text-ink-4" />
              </button>
            </div>
            <form onSubmit={handleSubmit} className="space-y-3">
              {creating && <StartFromSelect value={form.defaultKey} onChoose={startFrom} />}

              <div>
                <label htmlFor="template-name" className="block text-2xs text-ink-4 caps-label mb-1.5">
                  {t('templates.name')}
                </label>
                <input
                  id="template-name"
                  type="text"
                  required
                  autoFocus
                  value={form.name}
                  onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                  className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                />
              </div>

              <div>
                <label htmlFor="template-category" className="block text-2xs text-ink-4 caps-label mb-1.5">
                  {t('templates.category')}
                </label>
                <select
                  id="template-category"
                  value={form.category}
                  onChange={e => setForm(f => ({ ...f, category: e.target.value as MessageCategory }))}
                  className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                >
                  {ALL_CATEGORIES.map(c => (
                    <option key={c} value={c}>{CATEGORY_LABELS[c]}</option>
                  ))}
                </select>
              </div>

              <div>
                <span className="block text-2xs text-ink-4 caps-label mb-1.5">
                  {t('templates.channels')}
                </span>
                <div className="flex items-center gap-4">
                  {ALL_CHANNELS.map(ch => (
                    <label key={ch} className="flex items-center gap-1.5 text-xs text-ink-2 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={form.channels.includes(ch)}
                        onChange={() => toggleChannel(ch)}
                        className="rounded"
                      />
                      {CHANNEL_LABELS[ch]}
                    </label>
                  ))}
                </div>
              </div>

              {form.channels.includes('email') && (
                <div>
                  <label htmlFor="template-subject" className="block text-2xs text-ink-4 caps-label mb-1.5">
                    {t('templates.subject')}
                  </label>
                  <input
                    id="template-subject"
                    type="text"
                    value={form.subject}
                    onChange={e => setForm(f => ({ ...f, subject: e.target.value }))}
                    className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                  />
                </div>
              )}

              {form.channels.includes('email') && (
                <div>
                  <label htmlFor="template-body" className="block text-2xs text-ink-4 caps-label mb-1.5">
                    {t('templates.body')}
                  </label>
                  <textarea
                    id="template-body"
                    required
                    rows={4}
                    value={form.body}
                    onChange={e => setForm(f => ({ ...f, body: e.target.value }))}
                    placeholder={t('templates.bodyPlaceholder', { customer: '{{customer_name}}', order: '{{order_number}}' })}
                    className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 font-mono outline-none focus:ring-1 focus:ring-focus"
                  />
                  {editingTemplate && <p className="text-2xs text-ink-4 mt-1">{t('templates.designHint')}</p>}
                </div>
              )}

              {form.channels.some(ch => ch !== 'email') && (
                <div>
                  <label htmlFor="template-text" className="block text-2xs text-ink-4 caps-label mb-1.5">
                    {t('templates.textBody')}
                  </label>
                  <textarea
                    id="template-text"
                    rows={4}
                    value={form.textBody}
                    onChange={e => setForm(f => ({ ...f, textBody: e.target.value }))}
                    placeholder={t('templates.bodyPlaceholder', { customer: '{{customer_name}}', order: '{{order_number}}' })}
                    className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                  />
                </div>
              )}

              <p className="text-2xs text-ink-4">
                {t('templates.tokensHint', { token: '{{token}}', loop: '{{#items}}…{{/items}}', ifSet: '{{#sale_price}}…{{/sale_price}}', ifEmpty: '{{^sale_price}}…{{/sale_price}}' })}
              </p>

              {editingTemplate && (
                <div className="flex items-center justify-between">
                  <span className="text-2xs text-ink-4 caps-label">{t('templates.enabled')}</span>
                  <label className="relative flex items-center shrink-0 cursor-pointer">
                    <input
                      type="checkbox"
                      role="switch"
                      aria-label={t('templates.enabled')}
                      checked={form.enabled}
                      onChange={e => setForm(f => ({ ...f, enabled: e.target.checked }))}
                      className="sr-only peer"
                    />
                    <div className="w-8 h-4 dark:bg-night-700 bg-gray-300 rounded-full peer peer-checked:bg-accent-solid transition-colors" />
                    <div className="absolute left-0.5 top-0.5 w-3 h-3 bg-white rounded-full transition-all peer-checked:translate-x-4" />
                  </label>
                </div>
              )}

              {formError && (
                <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError}</p>
              )}

              <button
                type="submit"
                disabled={saving}
                className="w-full py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
              >
                {saving ? t('common:actions.saving') : editingTemplate ? t('common:actions.saveChanges') : t('templates.create')}
              </button>
            </form>
          </div>
        </Dialog>
      )}
      {confirmDialog}
    </div>
  )
}

import { useState } from 'react'
import { Search, Plus, Pencil, Trash2, X, Mail, MessageSquare, Phone } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import { TEMPLATES_PATH } from './messageCenterPaths'
import { useRoutedEditor, NEW_RECORD } from '../routing/useRoutedEditor'
import { useInitOnce } from '../routing/useInitOnce'
import { useMessageTemplates } from './hooks/useMessageTemplates'
import type { MessageTemplate, MessageChannel, MessageCategory } from './types'

const CATEGORY_LABELS: Record<MessageCategory, string> = {
  purchase: 'Purchase',
  order_status: 'Order status',
  stock_info: 'Stock info',
  tracking: 'Tracking',
  custom: 'Custom',
}

const CHANNEL_LABELS: Record<MessageChannel, string> = {
  email: 'Email',
  sms: 'SMS',
  whatsapp: 'WhatsApp',
}

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
  enabled: boolean
}

function emptyForm(): FormState {
  return { name: '', category: 'purchase', channels: [], subject: '', body: '', enabled: true }
}

function toForm(t: MessageTemplate): FormState {
  return { name: t.name, category: t.category, channels: t.channels, subject: t.subject, body: t.body, enabled: t.enabled }
}

export default function MessageTemplatesPanel() {
  const { token } = useAuth()
  const { templates, loading, error, create, update, remove } = useMessageTemplates(token ?? '')

  const [search, setSearch] = useState('')
  // Editor open state is the URL (/message-center/templates/<new|id>).
  const editor = useRoutedEditor(TEMPLATES_PATH)
  const creating = editor.editingId === NEW_RECORD
  const editingTemplate = editor.editingId && !creating ? templates.find(t => t.id === editor.editingId) ?? null : null
  const [form, setForm] = useState<FormState>(emptyForm())
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')
  const [actionError, setActionError] = useState('')

  const q = search.toLowerCase()
  const filtered = templates.filter(t => t.name.toLowerCase().includes(q) || t.body.toLowerCase().includes(q))

  const formReady = useInitOnce(editor.editingId, creating || !!editingTemplate, () => {
    setForm(editingTemplate ? toForm(editingTemplate) : emptyForm())
    setFormError('')
  })
  const openCreate = editor.openNew
  const openEdit = (t: MessageTemplate) => editor.openEdit(t.id)
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
      if (editingTemplate) {
        await update(editingTemplate.id, { ...form })
      } else {
        await create({ name: form.name, category: form.category, channels: form.channels, subject: form.subject, body: form.body })
      }
      closeForm()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete(t: MessageTemplate) {
    if (!window.confirm(`Delete template "${t.name}"? This cannot be undone.`)) return
    setActionError('')
    try {
      await remove(t.id)
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Unknown error')
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <div className="relative flex-1 max-w-sm">
          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-4" />
          <input
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder="Search templates..."
            className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg bg-raised border-line border text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        <button
          onClick={openCreate}
          className="ml-auto flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700 transition-colors"
        >
          <Plus size={13} /> New template
        </button>
      </div>

      <div className="flex items-center gap-6 px-5 py-2.5 border-b border-line bg-surface shrink-0">
        <span className="text-xs text-ink-4">{filtered.length} template{filtered.length === 1 ? '' : 's'}</span>
        {(error || actionError) && <span className="text-xs text-danger ml-auto">{error || actionError}</span>}
      </div>

      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Name</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Category</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Channels</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Body</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Status</th>
              <th className="px-3 py-2.5 w-16" />
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {loading && (
              <tr><td colSpan={6} className="px-4 py-6 text-center text-ink-4">Loading…</td></tr>
            )}
            {!loading && filtered.length === 0 && (
              <tr><td colSpan={6} className="px-4 py-6 text-center text-ink-4">No message templates yet</td></tr>
            )}
            {filtered.map(t => (
              <tr key={t.id} className="group hover:bg-hover/60 transition-colors">
                <td className="px-3 py-2.5 font-medium text-ink">{t.name}</td>
                <td className="px-3 py-2.5 text-ink-3">{CATEGORY_LABELS[t.category]}</td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1.5">
                    {t.channels.map(ch => {
                      const Icon = CHANNEL_ICONS[ch]
                      return <span key={ch} title={CHANNEL_LABELS[ch]}><Icon size={12} className="text-ink-3" /></span>
                    })}
                  </div>
                </td>
                <td className="px-3 py-2.5 max-w-[320px] truncate text-ink-3">{t.body}</td>
                <td className="px-3 py-2.5">
                  <span className={t.enabled ? 'text-success' : 'text-ink-4'}>
                    {t.enabled ? 'Enabled' : 'Disabled'}
                  </span>
                </td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button
                      onClick={() => openEdit(t)}
                      className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
                      title="Edit"
                    >
                      <Pencil size={12} className="text-ink-3" />
                    </button>
                    <button
                      onClick={() => handleDelete(t)}
                      className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
                      title="Delete"
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
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={closeForm} />
          <div className="relative w-full max-w-md bg-raised rounded-xl shadow-2xl border border-line p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-ink">
                {editingTemplate ? 'Edit message template' : 'New message template'}
              </h3>
              <button onClick={closeForm} aria-label="Close">
                <X size={14} className="text-ink-4" />
              </button>
            </div>
            <form onSubmit={handleSubmit} className="space-y-3">
              <div>
                <label htmlFor="template-name" className="block text-[10px] text-ink-4 uppercase tracking-wider mb-1.5">
                  Name
                </label>
                <input
                  id="template-name"
                  type="text"
                  required
                  autoFocus
                  value={form.name}
                  onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                  className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>

              <div>
                <label htmlFor="template-category" className="block text-[10px] text-ink-4 uppercase tracking-wider mb-1.5">
                  Category
                </label>
                <select
                  id="template-category"
                  value={form.category}
                  onChange={e => setForm(f => ({ ...f, category: e.target.value as MessageCategory }))}
                  className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                >
                  {ALL_CATEGORIES.map(c => (
                    <option key={c} value={c}>{CATEGORY_LABELS[c]}</option>
                  ))}
                </select>
              </div>

              <div>
                <span className="block text-[10px] text-ink-4 uppercase tracking-wider mb-1.5">
                  Channels
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
                  <label htmlFor="template-subject" className="block text-[10px] text-ink-4 uppercase tracking-wider mb-1.5">
                    Subject (email only)
                  </label>
                  <input
                    id="template-subject"
                    type="text"
                    value={form.subject}
                    onChange={e => setForm(f => ({ ...f, subject: e.target.value }))}
                    className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                  />
                </div>
              )}

              <div>
                <label htmlFor="template-body" className="block text-[10px] text-ink-4 uppercase tracking-wider mb-1.5">
                  Body
                </label>
                <textarea
                  id="template-body"
                  required
                  rows={4}
                  value={form.body}
                  onChange={e => setForm(f => ({ ...f, body: e.target.value }))}
                  placeholder="Hi {{customer_name}}, your order #{{order_number}} is confirmed."
                  className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                />
                <p className="text-[10px] text-ink-4 mt-1">
                  Use {'{{token}}'} placeholders -- available tokens depend on which event sends this template.
                </p>
              </div>

              {editingTemplate && (
                <div className="flex items-center justify-between">
                  <span className="text-[10px] text-ink-4 uppercase tracking-wider">Enabled</span>
                  <label className="relative flex items-center shrink-0 cursor-pointer">
                    <input
                      type="checkbox"
                      role="switch"
                      aria-label="Enabled"
                      checked={form.enabled}
                      onChange={e => setForm(f => ({ ...f, enabled: e.target.checked }))}
                      className="sr-only peer"
                    />
                    <div className="w-8 h-4 dark:bg-night-700 bg-gray-300 rounded-full peer peer-checked:bg-blue-600 transition-colors" />
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
                className="w-full py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
              >
                {saving ? 'Saving…' : editingTemplate ? 'Save changes' : 'Create template'}
              </button>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

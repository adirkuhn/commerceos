import { useState } from 'react'
import { Plus, Pencil, Trash2, X } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import { useAutomationRules } from './hooks/useAutomationRules'
import { useMessageTemplates } from './hooks/useMessageTemplates'
import type { AutomationRule, Condition, Operator, TriggerSubject, MessageChannel } from './types'

const TRIGGER_LABELS: Record<TriggerSubject, string> = {
  'pms.product.deleted': 'Product deleted',
  'storefront.order.placed': 'Storefront order placed',
  'pms.category.changed': 'Category changed',
  'pms.product.changed': 'Product changed',
  'payments.refund.manual_review_required': 'Refund needs manual review',
  'payments.refund.issued': 'Refund issued',
  'integrations.connection.connected': 'Integration connected',
  'oms.sales_order.status_changed': 'Order status changed',
  'inventory.stock.changed': 'Stock changed',
  'shipping.shipment.fulfillment_signal': 'Shipment status update',
}

const ALL_TRIGGERS = Object.keys(TRIGGER_LABELS) as TriggerSubject[]
const ALL_OPERATORS: Operator[] = ['eq', 'neq', 'in', 'gt', 'lt']
const OPERATOR_LABELS: Record<Operator, string> = { eq: 'equals', neq: 'not equals', in: 'is one of', gt: '>', lt: '<' }
const ALL_CHANNELS: MessageChannel[] = ['email', 'sms', 'whatsapp']

interface ConditionRow {
  field: string
  operator: Operator
  valueText: string
}

interface FormState {
  name: string
  triggerSubject: TriggerSubject
  templateId: string
  channelOverride: MessageChannel | ''
  conditions: ConditionRow[]
  enabled: boolean
}

function parsePrimitive(text: string): unknown {
  const trimmed = text.trim()
  if (trimmed === 'true') return true
  if (trimmed === 'false') return false
  if (trimmed !== '' && !Number.isNaN(Number(trimmed))) return Number(trimmed)
  return trimmed
}

function conditionToRow(c: Condition): ConditionRow {
  const valueText = Array.isArray(c.value) ? c.value.join(', ') : String(c.value)
  return { field: c.field, operator: c.operator, valueText }
}

function rowToCondition(row: ConditionRow): Condition {
  const value = row.operator === 'in' ? row.valueText.split(',').map(s => parsePrimitive(s)) : parsePrimitive(row.valueText)
  return { field: row.field, operator: row.operator, value }
}

function conditionSummary(c: Condition): string {
  const value = Array.isArray(c.value) ? c.value.join(', ') : String(c.value)
  return `${c.field} ${OPERATOR_LABELS[c.operator]} ${value}`
}

function emptyForm(): FormState {
  return { name: '', triggerSubject: ALL_TRIGGERS[0], templateId: '', channelOverride: '', conditions: [], enabled: true }
}

function toForm(r: AutomationRule): FormState {
  return {
    name: r.name, triggerSubject: r.trigger_subject, templateId: r.template_id,
    channelOverride: r.channel_override ?? '', conditions: r.conditions.map(conditionToRow), enabled: r.enabled,
  }
}

export default function AutomationRulesPanel() {
  const { token } = useAuth()
  const { rules, loading, error, create, update, remove } = useAutomationRules(token ?? '')
  const { templates } = useMessageTemplates(token ?? '')

  const [creating, setCreating] = useState(false)
  const [editingRule, setEditingRule] = useState<AutomationRule | null>(null)
  const [form, setForm] = useState<FormState>(emptyForm())
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')
  const [actionError, setActionError] = useState('')

  function templateName(id: string): string {
    return templates.find(t => t.id === id)?.name ?? id
  }

  function openCreate() {
    setForm({ ...emptyForm(), templateId: templates[0]?.id ?? '' })
    setFormError('')
    setCreating(true)
  }

  function openEdit(r: AutomationRule) {
    setForm(toForm(r))
    setFormError('')
    setEditingRule(r)
  }

  function closeForm() {
    setCreating(false)
    setEditingRule(null)
  }

  function addCondition() {
    setForm(f => ({ ...f, conditions: [...f.conditions, { field: '', operator: 'eq', valueText: '' }] }))
  }

  function updateCondition(index: number, patch: Partial<ConditionRow>) {
    setForm(f => ({ ...f, conditions: f.conditions.map((c, i) => (i === index ? { ...c, ...patch } : c)) }))
  }

  function removeCondition(index: number) {
    setForm(f => ({ ...f, conditions: f.conditions.filter((_, i) => i !== index) }))
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setSaving(true)
    setFormError('')
    try {
      const input = {
        name: form.name, trigger_subject: form.triggerSubject, template_id: form.templateId,
        channel_override: form.channelOverride || undefined,
        conditions: form.conditions.filter(c => c.field !== '').map(rowToCondition),
      }
      if (editingRule) {
        await update(editingRule.id, { ...input, enabled: form.enabled })
      } else {
        await create(input)
      }
      closeForm()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete(r: AutomationRule) {
    if (!window.confirm(`Delete rule "${r.name}"? This cannot be undone.`)) return
    setActionError('')
    try {
      await remove(r.id)
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Unknown error')
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-3 px-5 py-3 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <p className="text-xs dark:text-gray-500 text-gray-400">
          When an event fires and every condition matches, the chosen template is sent.
        </p>
        <button
          onClick={openCreate}
          disabled={templates.length === 0}
          title={templates.length === 0 ? 'Create a message template first' : undefined}
          className="ml-auto flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50 transition-colors"
        >
          <Plus size={13} /> New rule
        </button>
      </div>

      <div className="flex items-center gap-6 px-5 py-2.5 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-white shrink-0">
        <span className="text-xs dark:text-gray-500 text-gray-400">{rules.length} rule{rules.length === 1 ? '' : 's'}</span>
        {(error || actionError) && <span className="text-xs text-red-400 ml-auto">{error || actionError}</span>}
      </div>

      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
            <tr>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Name</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Trigger</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Conditions</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Template</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Status</th>
              <th className="px-3 py-2.5 w-16" />
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
            {loading && (
              <tr><td colSpan={6} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">Loading…</td></tr>
            )}
            {!loading && rules.length === 0 && (
              <tr><td colSpan={6} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">No automation rules yet</td></tr>
            )}
            {rules.map(r => (
              <tr key={r.id} className="group hover:dark:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors">
                <td className="px-3 py-2.5 font-medium dark:text-gray-100 text-gray-900">{r.name}</td>
                <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">{TRIGGER_LABELS[r.trigger_subject]}</td>
                <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">
                  {r.conditions.length === 0 ? 'Always' : r.conditions.map(conditionSummary).join(' AND ')}
                </td>
                <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">{templateName(r.template_id)}</td>
                <td className="px-3 py-2.5">
                  <span className={r.enabled ? 'text-green-500' : 'dark:text-gray-500 text-gray-400'}>
                    {r.enabled ? 'Enabled' : 'Disabled'}
                  </span>
                </td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button
                      onClick={() => openEdit(r)}
                      className="w-6 h-6 flex items-center justify-center rounded dark:hover:bg-[#2D3748] hover:bg-gray-200 transition-colors"
                      title="Edit"
                    >
                      <Pencil size={12} className="dark:text-gray-400 text-gray-500" />
                    </button>
                    <button
                      onClick={() => handleDelete(r)}
                      className="w-6 h-6 flex items-center justify-center rounded dark:hover:bg-[#2D3748] hover:bg-gray-200 transition-colors"
                      title="Delete"
                    >
                      <Trash2 size={12} className="dark:text-gray-400 text-gray-500 hover:text-red-400" />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {(creating || editingRule) && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={closeForm} />
          <div className="relative w-full max-w-lg dark:bg-[#1F2937] bg-white rounded-xl shadow-2xl border dark:border-[#2D3748] border-gray-200 p-5 max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">
                {editingRule ? 'Edit automation rule' : 'New automation rule'}
              </h3>
              <button onClick={closeForm} aria-label="Close">
                <X size={14} className="dark:text-gray-500 text-gray-400" />
              </button>
            </div>
            <form onSubmit={handleSubmit} className="space-y-3">
              <div>
                <label htmlFor="rule-name" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                  Name
                </label>
                <input
                  id="rule-name"
                  type="text"
                  required
                  autoFocus
                  value={form.name}
                  onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                  className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>

              <div>
                <label htmlFor="rule-trigger" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                  When this happens
                </label>
                <select
                  id="rule-trigger"
                  value={form.triggerSubject}
                  onChange={e => setForm(f => ({ ...f, triggerSubject: e.target.value as TriggerSubject }))}
                  className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                >
                  {ALL_TRIGGERS.map(t => (
                    <option key={t} value={t}>{TRIGGER_LABELS[t]}</option>
                  ))}
                </select>
              </div>

              <div>
                <div className="flex items-center justify-between mb-1.5">
                  <span className="text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider">
                    And these conditions match (optional)
                  </span>
                  <button type="button" onClick={addCondition} className="text-[10px] text-blue-500 hover:text-blue-400">
                    + Add condition
                  </button>
                </div>
                <div className="space-y-1.5">
                  {form.conditions.map((c, i) => (
                    <div key={i} className="flex items-center gap-1.5">
                      <input
                        type="text"
                        placeholder="field"
                        value={c.field}
                        onChange={e => updateCondition(i, { field: e.target.value })}
                        className="w-24 px-2 py-1.5 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                      />
                      <select
                        value={c.operator}
                        onChange={e => updateCondition(i, { operator: e.target.value as Operator })}
                        className="px-2 py-1.5 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                      >
                        {ALL_OPERATORS.map(op => (
                          <option key={op} value={op}>{OPERATOR_LABELS[op]}</option>
                        ))}
                      </select>
                      <input
                        type="text"
                        placeholder={c.operator === 'in' ? 'value1, value2' : 'value'}
                        value={c.valueText}
                        onChange={e => updateCondition(i, { valueText: e.target.value })}
                        className="flex-1 px-2 py-1.5 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                      />
                      <button type="button" onClick={() => removeCondition(i)} aria-label="Remove condition">
                        <X size={13} className="dark:text-gray-500 text-gray-400" />
                      </button>
                    </div>
                  ))}
                </div>
              </div>

              <div>
                <label htmlFor="rule-template" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                  Send this template
                </label>
                <select
                  id="rule-template"
                  required
                  value={form.templateId}
                  onChange={e => setForm(f => ({ ...f, templateId: e.target.value }))}
                  className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                >
                  <option value="" disabled>Select a template</option>
                  {templates.map(t => (
                    <option key={t.id} value={t.id}>{t.name}</option>
                  ))}
                </select>
              </div>

              <div>
                <label htmlFor="rule-channel-override" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                  Channel override (optional)
                </label>
                <select
                  id="rule-channel-override"
                  value={form.channelOverride}
                  onChange={e => setForm(f => ({ ...f, channelOverride: e.target.value as MessageChannel | '' }))}
                  className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                >
                  <option value="">Use template's own channels</option>
                  {ALL_CHANNELS.map(ch => (
                    <option key={ch} value={ch}>{ch}</option>
                  ))}
                </select>
              </div>

              {editingRule && (
                <div className="flex items-center justify-between">
                  <span className="text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider">Enabled</span>
                  <label className="relative flex items-center shrink-0 cursor-pointer">
                    <input
                      type="checkbox"
                      role="switch"
                      aria-label="Enabled"
                      checked={form.enabled}
                      onChange={e => setForm(f => ({ ...f, enabled: e.target.checked }))}
                      className="sr-only peer"
                    />
                    <div className="w-8 h-4 dark:bg-[#2D3748] bg-gray-300 rounded-full peer peer-checked:bg-blue-600 transition-colors" />
                    <div className="absolute left-0.5 top-0.5 w-3 h-3 bg-white rounded-full transition-all peer-checked:translate-x-4" />
                  </label>
                </div>
              )}

              {formError && (
                <p className="text-xs text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError}</p>
              )}

              <button
                type="submit"
                disabled={saving}
                className="w-full py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
              >
                {saving ? 'Saving…' : editingRule ? 'Save changes' : 'Create rule'}
              </button>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

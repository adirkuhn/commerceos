import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import { Plus, Pencil, Trash2, X } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import { RULES_PATH } from './messageCenterPaths'
import { useRoutedEditor, NEW_RECORD } from '../routing/useRoutedEditor'
import { useInitOnce } from '../routing/useInitOnce'
import { useAutomationRules } from './hooks/useAutomationRules'
import { useMessageTemplates } from './hooks/useMessageTemplates'
import type { AutomationRule, Condition, Operator, TriggerSubject, MessageChannel } from './types'
import { Dialog } from '../components/common/Dialog'
import { useConfirm } from '../components/common/useConfirm'
import { triggerLabels } from './triggerLabels'
import { ExtraRecipientsField } from './ExtraRecipientsField'
import { parseEmailList } from './emailList'
import { apiErrorText } from '../api/apiErrorText'

type T = TFunction<'messages'>


// Triggers addressed to the tenant's own team (backend
// domain/messagecenter.AudienceFor), not to the order's customer.
const STAFF_TRIGGERS: TriggerSubject[] = ['inventory.stock.close_to_expiry_reported']

const ALL_TRIGGERS: TriggerSubject[] = [
  'pms.product.deleted', 'storefront.order.placed', 'pms.category.changed', 'pms.product.changed',
  'payments.refund.manual_review_required', 'payments.refund.issued', 'integrations.connection.connected',
  'oms.sales_order.status_changed', 'inventory.stock.changed', 'shipping.shipment.fulfillment_signal',
  'inventory.stock.close_to_expiry_reported', 'oms.sales_order.created',
]
const ALL_OPERATORS: Operator[] = ['eq', 'neq', 'in', 'gt', 'lt']
const operatorLabels = (t: T): Record<Operator, string> => ({ eq: t('operator.eq'), neq: t('operator.neq'), in: t('operator.in'), gt: '>', lt: '<' })
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
  extraRecipientsText: string
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

function conditionSummary(t: T, c: Condition): string {
  const value = Array.isArray(c.value) ? c.value.join(', ') : String(c.value)
  return `${c.field} ${operatorLabels(t)[c.operator]} ${value}`
}

function emptyForm(): FormState {
  return { name: '', triggerSubject: ALL_TRIGGERS[0], templateId: '', channelOverride: '', conditions: [], extraRecipientsText: '', enabled: true }
}

function toForm(r: AutomationRule): FormState {
  return {
    name: r.name, triggerSubject: r.trigger_subject, templateId: r.template_id,
    channelOverride: r.channel_override ?? '', conditions: r.conditions.map(conditionToRow),
    extraRecipientsText: (r.extra_recipients ?? []).join('\n'), enabled: r.enabled,
  }
}

export default function AutomationRulesPanel() {
  const { confirm, confirmDialog } = useConfirm()
  const { token } = useAuth()
  const { t } = useTranslation(['messages', 'common'])
  const TRIGGER_LABELS = triggerLabels(t)
  const OPERATOR_LABELS = operatorLabels(t)
  const CHANNEL_LABELS: Record<MessageChannel, string> = { email: t('channel.email'), sms: t('channel.sms'), whatsapp: t('channel.whatsapp') }
  const { rules, loading, error, create, update, remove } = useAutomationRules(token ?? '')
  const { templates, loading: templatesLoading } = useMessageTemplates(token ?? '')

  // Editor open state is the URL (/message-center/rules/<new|id>).
  const editor = useRoutedEditor(RULES_PATH)
  const creating = editor.editingId === NEW_RECORD
  const editingRule = editor.editingId && !creating ? rules.find(r => r.id === editor.editingId) ?? null : null
  const [form, setForm] = useState<FormState>(emptyForm())
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')
  const [actionError, setActionError] = useState('')

  function templateName(id: string): string {
    return templates.find(tpl => tpl.id === id)?.name ?? id
  }

  // A new rule defaults to the first template, so it waits for templates.
  const formReady = useInitOnce(editor.editingId, creating ? !templatesLoading : !!editingRule, () => {
    setForm(editingRule ? toForm(editingRule) : { ...emptyForm(), templateId: templates[0]?.id ?? '' })
    setFormError('')
  })
  const openCreate = editor.openNew
  const openEdit = (r: AutomationRule) => editor.openEdit(r.id)
  const closeForm = editor.close

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
        extra_recipients: parseEmailList(form.extraRecipientsText),
      }
      if (editingRule) {
        await update(editingRule.id, { ...input, enabled: form.enabled })
      } else {
        await create(input)
      }
      closeForm()
    } catch (err) {
      setFormError(apiErrorText(err) ?? t('common:status.error'))
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete(r: AutomationRule) {
    if (!(await confirm({ title: t('rules.deleteTitle', { name: r.name }), message: t('rules.deleteMessage'), confirmLabel: t('rules.deleteConfirm'), destructive: true }))) return
    setActionError('')
    try {
      await remove(r.id)
    } catch (err) {
      setActionError(apiErrorText(err) ?? t('common:status.error'))
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <p className="text-xs text-ink-4">
          {t('rules.intro')}
        </p>
        <button
          onClick={openCreate}
          disabled={templates.length === 0}
          title={templates.length === 0 ? t('rules.needTemplate') : undefined}
          className="ml-auto flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-accent-solid text-white hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
        >
          <Plus size={13} /> {t('rules.new')}
        </button>
      </div>

      <div className="flex items-center gap-6 px-5 py-2.5 border-b border-line bg-surface shrink-0">
        <span className="text-xs text-ink-4">{t('rules.count', { count: rules.length })}</span>
        {(error || actionError) && <span className="text-xs text-danger ml-auto">{error || actionError}</span>}
      </div>

      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{t('rules.colName')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{t('rules.colTrigger')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{t('rules.colConditions')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{t('rules.colTemplate')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{t('rules.colStatus')}</th>
              <th className="px-3 py-2.5 w-16" />
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {loading && (
              <tr><td colSpan={6} className="px-4 py-6 text-center text-ink-4">{t('common:status.loading')}</td></tr>
            )}
            {!loading && rules.length === 0 && (
              <tr><td colSpan={6} className="px-4 py-6 text-center text-ink-4">{t('rules.empty')}</td></tr>
            )}
            {rules.map(r => (
              <tr key={r.id} className="group hover:bg-hover/60 transition-colors">
                <td className="px-3 py-2.5 font-medium text-ink">{r.name}</td>
                <td className="px-3 py-2.5 text-ink-3">{TRIGGER_LABELS[r.trigger_subject]}</td>
                <td className="px-3 py-2.5 text-ink-3">
                  {r.conditions.length === 0 ? t('rules.always') : r.conditions.map(c => conditionSummary(t, c)).join(t('rules.and'))}
                </td>
                <td className="px-3 py-2.5 text-ink-3">{templateName(r.template_id)}</td>
                <td className="px-3 py-2.5">
                  <span className={r.enabled ? 'text-success' : 'text-ink-4'}>
                    {r.enabled ? t('common:status.enabled') : t('common:status.disabled')}
                  </span>
                </td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity">
                    <button aria-label={t('common:actions.edit')}
                      onClick={() => openEdit(r)}
                      className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
                      title={t('common:actions.edit')}
                    >
                      <Pencil size={12} className="text-ink-3" />
                    </button>
                    <button aria-label={t('common:actions.delete')}
                      onClick={() => handleDelete(r)}
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
          <div className="relative w-full max-w-lg bg-raised rounded-xl shadow-2xl border border-line p-5 max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-ink">
                {editingRule ? t('rules.editTitle') : t('rules.newTitle')}
              </h3>
              <button onClick={closeForm} aria-label={t('common:actions.close')}>
                <X size={14} className="text-ink-4" />
              </button>
            </div>
            <form onSubmit={handleSubmit} className="space-y-3">
              <div>
                <label htmlFor="rule-name" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                  {t('rules.name')}
                </label>
                <input
                  id="rule-name"
                  type="text"
                  required
                  autoFocus
                  value={form.name}
                  onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                  className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                />
              </div>

              <div>
                <label htmlFor="rule-trigger" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                  {t('rules.when')}
                </label>
                <select
                  id="rule-trigger"
                  value={form.triggerSubject}
                  onChange={e => setForm(f => ({ ...f, triggerSubject: e.target.value as TriggerSubject }))}
                  className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                >
                  {ALL_TRIGGERS.map(tr => (
                    <option key={tr} value={tr}>{TRIGGER_LABELS[tr]}</option>
                  ))}
                </select>
                {STAFF_TRIGGERS.includes(form.triggerSubject) && (
                  <p className="text-2xs text-ink-4 mt-1">{t('rules.staffAudience')}</p>
                )}
                {form.triggerSubject === 'oms.sales_order.created' && (
                  <p className="text-2xs text-ink-4 mt-1">{t('rules.orderCreatedHint')}</p>
                )}
              </div>

              <div>
                <div className="flex items-center justify-between mb-1.5">
                  <span className="text-2xs text-ink-4 uppercase tracking-wider">
                    {t('rules.conditions')}
                  </span>
                  <button type="button" onClick={addCondition} className="text-2xs text-accent hover:text-accent">
                    {t('rules.addCondition')}
                  </button>
                </div>
                <div className="space-y-1.5">
                  {form.conditions.map((c, i) => (
                    <div key={i} className="flex items-center gap-1.5">
                      <input
                        type="text"
                        placeholder={t('rules.field')}
                        value={c.field}
                        onChange={e => updateCondition(i, { field: e.target.value })}
                        className="w-24 px-2 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                      />
                      <select
                        value={c.operator}
                        onChange={e => updateCondition(i, { operator: e.target.value as Operator })}
                        className="px-2 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                      >
                        {ALL_OPERATORS.map(op => (
                          <option key={op} value={op}>{OPERATOR_LABELS[op]}</option>
                        ))}
                      </select>
                      <input
                        type="text"
                        placeholder={c.operator === 'in' ? t('rules.values') : t('rules.value')}
                        value={c.valueText}
                        onChange={e => updateCondition(i, { valueText: e.target.value })}
                        className="flex-1 px-2 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                      />
                      <button type="button" onClick={() => removeCondition(i)} aria-label={t('rules.removeCondition')}>
                        <X size={13} className="text-ink-4" />
                      </button>
                    </div>
                  ))}
                </div>
              </div>

              <div>
                <label htmlFor="rule-template" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                  {t('rules.template')}
                </label>
                <select
                  id="rule-template"
                  required
                  value={form.templateId}
                  onChange={e => setForm(f => ({ ...f, templateId: e.target.value }))}
                  className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                >
                  <option value="" disabled>{t('rules.selectTemplate')}</option>
                  {templates.map(tpl => (
                    <option key={tpl.id} value={tpl.id}>{tpl.name}</option>
                  ))}
                </select>
              </div>

              <div>
                <label htmlFor="rule-channel-override" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                  {t('rules.channelOverride')}
                </label>
                <select
                  id="rule-channel-override"
                  value={form.channelOverride}
                  onChange={e => setForm(f => ({ ...f, channelOverride: e.target.value as MessageChannel | '' }))}
                  className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                >
                  <option value="">{t('rules.templateChannels')}</option>
                  {ALL_CHANNELS.map(ch => (
                    <option key={ch} value={ch}>{CHANNEL_LABELS[ch]}</option>
                  ))}
                </select>
              </div>

              <ExtraRecipientsField
                value={form.extraRecipientsText}
                onChange={text => setForm(f => ({ ...f, extraRecipientsText: text }))}
              />

              {editingRule && (
                <div className="flex items-center justify-between">
                  <span className="text-2xs text-ink-4 uppercase tracking-wider">{t('rules.enabled')}</span>
                  <label className="relative flex items-center shrink-0 cursor-pointer">
                    <input
                      type="checkbox"
                      role="switch"
                      aria-label={t('rules.enabled')}
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
                {saving ? t('common:actions.saving') : editingRule ? t('common:actions.saveChanges') : t('rules.create')}
              </button>
            </form>
          </div>
        </Dialog>
      )}
      {confirmDialog}
    </div>
  )
}

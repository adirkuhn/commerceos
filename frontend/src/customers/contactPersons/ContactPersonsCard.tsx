import { useState } from 'react'
import { Edit3, Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { apiErrorText } from '../../api/apiErrorText'
import { useConfirm } from '../../components/common/useConfirm'
import type { ContactPerson, ContactPersonInput } from '../types'

interface ContactPersonsCardProps {
  persons: ContactPerson[]
  error: string | null
  onAdd: (input: ContactPersonInput) => Promise<void>
  onUpdate: (id: string, input: ContactPersonInput) => Promise<void>
  onRemove: (id: string) => Promise<void>
}

const personName = (p: ContactPersonInput) => [p.first_name, p.last_name].filter(Boolean).join(' ')

// A company customer's contact persons, the main one first: list, add,
// edit, remove.
export function ContactPersonsCard({ persons, error, onAdd, onUpdate, onRemove }: ContactPersonsCardProps) {
  const { t } = useTranslation('customers')
  const { confirm, confirmDialog } = useConfirm()
  // 'new', a person's id, or null when no form is open.
  const [editing, setEditing] = useState<string | null>(null)

  async function remove(p: ContactPerson) {
    const ok = await confirm({ title: t('contactPersons.removeTitle', { name: personName(p) }), message: t('contactPersons.removeMessage'), confirmLabel: t('contactPersons.remove'), destructive: true })
    if (!ok) return
    await onRemove(p.id)
    setEditing(null)
  }

  return (
    <div className="rounded-xl border border-line p-4 space-y-3 text-xs">
      <h3 className="text-xs font-semibold text-ink">{t('contactPersons.title')}</h3>
      {error && <p role="alert" className="text-danger">{error}</p>}
      {persons.length === 0 && editing !== 'new' && <p className="text-ink-4">{t('contactPersons.empty')}</p>}
      <ul className="space-y-2">
        {persons.map(p => (
          <li key={p.id} className="rounded-lg border border-line/70 px-3 py-2">
            {editing === p.id ? (
              <ContactPersonForm initial={p} onCancel={() => setEditing(null)}
                onSave={async input => { await onUpdate(p.id, input); setEditing(null) }} onRemove={() => remove(p)} />
            ) : (
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  {p.role && <p className="text-2xs caps-label text-ink-4">{p.role}</p>}
                  <p className="text-ink truncate">{personName(p)}</p>
                  {(p.email || p.phone) && <p className="text-2xs text-ink-3 truncate">{[p.email, p.phone].filter(Boolean).join(' · ')}</p>}
                </div>
                <button type="button" onClick={() => setEditing(p.id)} aria-label={t('contactPersons.edit', { name: personName(p) })}
                  title={t('contactPersons.edit', { name: personName(p) })}
                  className="w-6 h-6 shrink-0 flex items-center justify-center rounded hover:bg-hover"><Edit3 size={12} className="text-ink-3" /></button>
              </div>
            )}
          </li>
        ))}
      </ul>
      {editing === 'new' ? (
        <div className="rounded-lg border border-line/70 px-3 py-2">
          <ContactPersonForm onCancel={() => setEditing(null)} onSave={async input => { await onAdd(input); setEditing(null) }} />
        </div>
      ) : (
        <button type="button" onClick={() => setEditing('new')}
          className="w-full flex items-center justify-center gap-1.5 px-3 py-1.5 rounded-lg border border-line text-xs font-medium text-ink-2 hover:bg-hover">
          <Plus size={12} /> {t('contactPersons.add')}
        </button>
      )}
      {confirmDialog}
    </div>
  )
}

const FIELDS = ['first_name', 'last_name', 'role', 'email', 'phone'] as const
type Field = typeof FIELDS[number]

function ContactPersonForm({ initial, onSave, onCancel, onRemove }: {
  initial?: ContactPerson
  onSave: (input: ContactPersonInput) => Promise<void>
  onCancel: () => void
  onRemove?: () => void
}) {
  const { t } = useTranslation(['customers', 'common'])
  const [form, setForm] = useState<Record<Field, string>>(() => ({
    first_name: initial?.first_name ?? '', last_name: initial?.last_name ?? '', role: initial?.role ?? '', email: initial?.email ?? '', phone: initial?.phone ?? '',
  }))
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const named = form.first_name.trim() !== '' || form.last_name.trim() !== ''

  async function save() {
    setSaving(true)
    setError('')
    try {
      await onSave(Object.fromEntries(FIELDS.map(f => [f, form[f].trim()])) as Record<Field, string>)
    } catch (err) {
      setError(apiErrorText(err) ?? t('contactPersons.failed'))
    } finally {
      setSaving(false)
    }
  }

  const input = 'mt-1 w-full px-2.5 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink outline-none focus:ring-1 focus:ring-focus normal-case'
  const type = (f: Field) => (f === 'email' ? 'email' : f === 'phone' ? 'tel' : 'text')
  return (
    <div className="space-y-2">
      <div className="grid grid-cols-2 gap-2">
        {FIELDS.map(f => (
          <label key={f} className={`block text-2xs font-medium text-ink-4 caps-label ${f === 'email' ? 'col-span-2' : ''}`}>
            {t(`contactPersons.fields.${f}`)}
            <input value={form[f]} type={type(f)} autoFocus={f === 'first_name'} className={input}
              onChange={e => setForm(prev => ({ ...prev, [f]: e.target.value }))} />
          </label>
        ))}
      </div>
      {error && <p role="alert" className="text-danger">{error}</p>}
      <div className="flex items-center gap-2">
        <button type="button" onClick={save} disabled={!named || saving}
          className="px-2.5 py-1 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-40">
          {t('common:actions.save')}
        </button>
        <button type="button" onClick={onCancel} className="px-2.5 py-1 rounded-lg border border-line text-xs text-ink-2 hover:bg-hover">
          {t('common:actions.cancel')}
        </button>
        {onRemove && (
          <button type="button" onClick={onRemove} className="ml-auto px-2.5 py-1 rounded-lg text-xs text-danger hover:bg-hover">
            {t('contactPersons.remove')}
          </button>
        )}
      </div>
    </div>
  )
}

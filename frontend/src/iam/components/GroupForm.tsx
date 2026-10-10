import { useState, useEffect, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import type { Group, CreateGroupPayload, UpdateGroupPayload } from '../types'
import { apiErrorText } from '../../api/apiErrorText'

interface Props {
  group?: Group | null
  onSubmit: (payload: CreateGroupPayload | UpdateGroupPayload) => Promise<void>
  onCancel: () => void
}

const inputClass = 'block w-full mt-1 px-3 py-1.5 text-xs rounded-lg bg-sunken border border-line text-ink outline-none focus:ring-1 focus:ring-focus'

export function GroupForm({ group, onSubmit, onCancel }: Props) {
  const { t } = useTranslation(['iam', 'common'])
  const [name, setName] = useState(group?.name ?? '')
  const [description, setDescription] = useState(group?.description ?? '')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    setName(group?.name ?? '')
    setDescription(group?.description ?? '')
  }, [group])

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (!name.trim()) {
      setError(t('groups.nameRequired'))
      return
    }
    setSubmitting(true)
    setError(null)
    try {
      await onSubmit({ name: name.trim(), description: description.trim() })
    } catch (e) {
      setError(apiErrorText(e) ?? t('unexpectedError'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-3 max-w-[400px]">
      <h3 className="text-xs font-semibold text-ink">{group ? t('groups.editTitle') : t('groups.newTitle')}</h3>

      {error && <p className="text-xs text-danger">{error}</p>}

      <label className="block text-xs text-ink-3">
        {t('groups.nameField')}
        <input
          type="text"
          value={name}
          onChange={e => setName(e.target.value)}
          maxLength={100}
          required
          className={inputClass}
        />
      </label>

      <label className="block text-xs text-ink-3">
        {t('common:fields.description')}
        <textarea
          value={description}
          onChange={e => setDescription(e.target.value)}
          maxLength={500}
          rows={3}
          className={inputClass}
        />
      </label>

      <div className="flex gap-2">
        <button
          type="submit"
          disabled={submitting}
          className="px-3 py-1.5 text-xs rounded-lg bg-accent-solid text-white hover:bg-accent-solid-hover disabled:opacity-50 font-medium"
        >
          {submitting ? t('common:actions.saving') : group ? t('groups.update') : t('common:actions.create')}
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="px-3 py-1.5 text-xs rounded-lg border border-line text-ink-2 hover:bg-hover"
        >
          {t('common:actions.cancel')}
        </button>
      </div>
    </form>
  )
}

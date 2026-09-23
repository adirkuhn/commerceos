import { useState, useEffect, type FormEvent } from 'react'
import type { Group, CreateGroupPayload, UpdateGroupPayload } from '../types'

interface Props {
  group?: Group | null
  onSubmit: (payload: CreateGroupPayload | UpdateGroupPayload) => Promise<void>
  onCancel: () => void
}

const inputClass = 'block w-full mt-1 px-3 py-1.5 text-xs rounded-lg bg-sunken border border-line text-ink outline-none focus:ring-1 focus:ring-focus'

export function GroupForm({ group, onSubmit, onCancel }: Props) {
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
      setError('Name is required')
      return
    }
    setSubmitting(true)
    setError(null)
    try {
      await onSubmit({ name: name.trim(), description: description.trim() })
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unexpected error')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-3 max-w-[400px]">
      <h3 className="text-xs font-semibold text-ink">{group ? 'Edit group' : 'New group'}</h3>

      {error && <p className="text-xs text-danger">{error}</p>}

      <label className="block text-xs text-ink-3">
        Name *
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
        Description
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
          {submitting ? 'Saving...' : group ? 'Update' : 'Create'}
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="px-3 py-1.5 text-xs rounded-lg border border-line text-ink-2 hover:bg-hover"
        >
          Cancel
        </button>
      </div>
    </form>
  )
}

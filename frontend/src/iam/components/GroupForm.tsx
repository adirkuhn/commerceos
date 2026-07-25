import { useState, useEffect, type FormEvent } from 'react'
import type { Group, CreateGroupPayload, UpdateGroupPayload } from '../types'

interface Props {
  group?: Group | null
  onSubmit: (payload: CreateGroupPayload | UpdateGroupPayload) => Promise<void>
  onCancel: () => void
}

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
    <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '12px', maxWidth: '400px' }}>
      <h3>{group ? 'Edit Group' : 'New Group'}</h3>

      {error && <p style={{ color: 'red' }}>{error}</p>}

      <label>
        Name *
        <input
          type="text"
          value={name}
          onChange={e => setName(e.target.value)}
          maxLength={100}
          required
          style={{ display: 'block', width: '100%', marginTop: '4px' }}
        />
      </label>

      <label>
        Description
        <textarea
          value={description}
          onChange={e => setDescription(e.target.value)}
          maxLength={500}
          rows={3}
          style={{ display: 'block', width: '100%', marginTop: '4px' }}
        />
      </label>

      <div style={{ display: 'flex', gap: '8px' }}>
        <button type="submit" disabled={submitting}>
          {submitting ? 'Saving...' : group ? 'Update' : 'Create'}
        </button>
        <button type="button" onClick={onCancel}>Cancel</button>
      </div>
    </form>
  )
}

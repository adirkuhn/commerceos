import { useState, type FormEvent } from 'react'

interface Props {
  groupId: string
  members: string[]
  loading: boolean
  error: string | null
  onAdd: (userId: string) => Promise<void>
  onRemove: (userId: string) => Promise<void>
}

export function MemberList({ members, loading, error, onAdd, onRemove }: Props) {
  const [newUserId, setNewUserId] = useState('')
  const [adding, setAdding] = useState(false)
  const [addError, setAddError] = useState<string | null>(null)

  const handleAdd = async (e: FormEvent) => {
    e.preventDefault()
    const uid = newUserId.trim()
    if (!uid) return
    setAdding(true)
    setAddError(null)
    try {
      await onAdd(uid)
      setNewUserId('')
    } catch (e) {
      setAddError(e instanceof Error ? e.message : 'Failed to add member')
    } finally {
      setAdding(false)
    }
  }

  if (loading) return <p>Loading members...</p>
  if (error) return <p style={{ color: 'red' }}>Error: {error}</p>

  return (
    <div>
      <h4>Members ({members.length})</h4>

      <form onSubmit={handleAdd} style={{ display: 'flex', gap: '8px', marginBottom: '12px' }}>
        <input
          type="text"
          value={newUserId}
          onChange={e => setNewUserId(e.target.value)}
          placeholder="User ID"
          style={{ flex: 1 }}
        />
        <button type="submit" disabled={adding || !newUserId.trim()}>
          {adding ? 'Adding...' : 'Add'}
        </button>
      </form>
      {addError && <p style={{ color: 'red' }}>{addError}</p>}

      {members.length === 0 ? (
        <p style={{ color: '#999' }}>No members.</p>
      ) : (
        <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
          {members.map(uid => (
            <li key={uid} style={{ display: 'flex', justifyContent: 'space-between', padding: '6px 0', borderBottom: '1px solid #eee' }}>
              <span style={{ fontFamily: 'monospace' }}>{uid}</span>
              <button onClick={() => onRemove(uid)} style={{ color: 'red', background: 'none', border: 'none', cursor: 'pointer' }}>
                Remove
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

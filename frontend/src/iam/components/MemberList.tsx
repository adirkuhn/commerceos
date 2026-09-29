import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'

interface Props {
  groupId: string
  members: string[]
  loading: boolean
  error: string | null
  onAdd: (userId: string) => Promise<void>
  onRemove: (userId: string) => Promise<void>
}

export function MemberList({ members, loading, error, onAdd, onRemove }: Props) {
  const { t } = useTranslation(['iam', 'common'])
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
      setAddError(e instanceof Error ? e.message : t('members.addFailed'))
    } finally {
      setAdding(false)
    }
  }

  if (loading) return <p>{t('members.loading')}</p>
  if (error) return <p style={{ color: 'red' }}>{t('errorPrefix', { error })}</p>

  return (
    <div>
      <h4>{t('members.title', { count: members.length })}</h4>

      <form onSubmit={handleAdd} style={{ display: 'flex', gap: '8px', marginBottom: '12px' }}>
        <input
          type="text"
          value={newUserId}
          onChange={e => setNewUserId(e.target.value)}
          placeholder={t('members.userId')}
          style={{ flex: 1 }}
        />
        <button type="submit" disabled={adding || !newUserId.trim()}>
          {adding ? t('members.adding') : t('common:actions.add')}
        </button>
      </form>
      {addError && <p style={{ color: 'red' }}>{addError}</p>}

      {members.length === 0 ? (
        <p style={{ color: '#999' }}>{t('members.none')}</p>
      ) : (
        <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
          {members.map(uid => (
            <li key={uid} style={{ display: 'flex', justifyContent: 'space-between', padding: '6px 0', borderBottom: '1px solid #eee' }}>
              <span style={{ fontFamily: 'monospace' }}>{uid}</span>
              <button onClick={() => onRemove(uid)} style={{ color: 'red', background: 'none', border: 'none', cursor: 'pointer' }}>
                {t('common:actions.remove')}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

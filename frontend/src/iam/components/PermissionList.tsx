import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'

interface Props {
  permissions: string[]
  loading: boolean
  error: string | null
  onAssign: (rule: string) => Promise<void>
  onRevoke: (rule: string) => Promise<void>
}

const RULE_PATTERN = /^[a-z]+\.[a-z_]+\.[a-z_]+$/

export function PermissionList({ permissions, loading, error, onAssign, onRevoke }: Props) {
  const { t } = useTranslation(['iam', 'common'])
  const [newRule, setNewRule] = useState('')
  const [assigning, setAssigning] = useState(false)
  const [assignError, setAssignError] = useState<string | null>(null)

  const isValidRule = RULE_PATTERN.test(newRule.trim())

  const handleAssign = async (e: FormEvent) => {
    e.preventDefault()
    const rule = newRule.trim()
    if (!isValidRule) {
      setAssignError(t('permissions.format'))
      return
    }
    setAssigning(true)
    setAssignError(null)
    try {
      await onAssign(rule)
      setNewRule('')
    } catch (e) {
      setAssignError(e instanceof Error ? e.message : t('permissions.assignFailed'))
    } finally {
      setAssigning(false)
    }
  }

  if (loading) return <p>{t('permissions.loading')}</p>
  if (error) return <p style={{ color: 'red' }}>{t('errorPrefix', { error })}</p>

  return (
    <div>
      <h4>{t('permissions.title', { count: permissions.length })}</h4>

      <form onSubmit={handleAssign} style={{ display: 'flex', gap: '8px', marginBottom: '12px' }}>
        <input
          type="text"
          value={newRule}
          onChange={e => setNewRule(e.target.value)}
          placeholder={'pms.product.create'}
          style={{ flex: 1, fontFamily: 'monospace' }}
        />
        <button type="submit" disabled={assigning || !newRule.trim()}>
          {assigning ? t('permissions.assigning') : t('groups.assign')}
        </button>
      </form>
      {assignError && <p style={{ color: 'red', fontSize: '0.85em' }}>{assignError}</p>}

      {permissions.length === 0 ? (
        <p style={{ color: '#999' }}>{t('permissions.none')}</p>
      ) : (
        <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
          {permissions.map(rule => (
            <li key={rule} style={{ display: 'flex', justifyContent: 'space-between', padding: '6px 0', borderBottom: '1px solid #eee' }}>
              <code style={{ fontSize: '0.9em' }}>{rule}</code>
              <button onClick={() => onRevoke(rule)} style={{ color: 'red', background: 'none', border: 'none', cursor: 'pointer' }}>
                {t('groups.revoke')}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

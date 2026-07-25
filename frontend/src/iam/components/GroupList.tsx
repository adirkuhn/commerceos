import type { Group } from '../types'

interface Props {
  groups: Group[]
  loading: boolean
  error: string | null
  onSelect: (group: Group) => void
  onDelete: (id: string) => void
  onCreateClick: () => void
}

export function GroupList({ groups, loading, error, onSelect, onDelete, onCreateClick }: Props) {
  if (loading) return <p>Loading groups...</p>
  if (error) return <p style={{ color: 'red' }}>Error: {error}</p>

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2>Groups</h2>
        <button onClick={onCreateClick}>+ New Group</button>
      </div>
      {groups.length === 0 ? (
        <p>No groups yet.</p>
      ) : (
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr>
              <th style={{ textAlign: 'left', padding: '8px' }}>Name</th>
              <th style={{ textAlign: 'left', padding: '8px' }}>Description</th>
              <th style={{ padding: '8px' }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {groups.map(group => (
              <tr key={group.id} style={{ borderTop: '1px solid #eee' }}>
                <td style={{ padding: '8px' }}>
                  <button
                    onClick={() => onSelect(group)}
                    style={{ background: 'none', border: 'none', cursor: 'pointer', textDecoration: 'underline' }}
                  >
                    {group.name}
                  </button>
                </td>
                <td style={{ padding: '8px', color: '#666' }}>{group.description}</td>
                <td style={{ padding: '8px', textAlign: 'center' }}>
                  <button onClick={() => onDelete(group.id)} style={{ color: 'red' }}>
                    Delete
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}

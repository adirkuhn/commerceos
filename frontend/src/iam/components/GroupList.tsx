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
  if (loading) return <p className="text-xs text-ink-3">Loading groups...</p>
  if (error) return <p className="text-xs text-danger">Error: {error}</p>

  return (
    <div>
      <div className="flex items-center justify-between mb-3">
        <h2 className="text-xs font-semibold text-ink">Groups</h2>
        <button
          onClick={onCreateClick}
          className="px-2.5 py-1 text-xs rounded-lg bg-blue-600 text-white hover:bg-blue-700 font-medium"
        >
          + New Group
        </button>
      </div>
      {groups.length === 0 ? (
        <p className="text-xs text-ink-4">No groups yet.</p>
      ) : (
        <table className="w-full text-xs">
          <thead>
            <tr className="border-b border-line">
              <th className="text-left py-2 font-medium text-ink-4 uppercase tracking-wider text-[10px]">Name</th>
              <th className="text-left py-2 font-medium text-ink-4 uppercase tracking-wider text-[10px]">Description</th>
              <th className="py-2" />
            </tr>
          </thead>
          <tbody>
            {groups.map(group => (
              <tr key={group.id} className="border-t border-line/60">
                <td className="py-2 pr-2">
                  <button
                    onClick={() => onSelect(group)}
                    className="text-accent hover:text-accent/80 underline underline-offset-2"
                  >
                    {group.name}
                  </button>
                </td>
                <td className="py-2 pr-2 text-ink-3">{group.description}</td>
                <td className="py-2 text-right">
                  <button onClick={() => onDelete(group.id)} className="text-danger hover:text-red-300">
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

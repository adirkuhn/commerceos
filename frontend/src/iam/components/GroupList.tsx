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
  if (loading) return <p className="text-xs dark:text-gray-400 text-gray-500">Loading groups...</p>
  if (error) return <p className="text-xs text-red-400">Error: {error}</p>

  return (
    <div>
      <div className="flex items-center justify-between mb-3">
        <h2 className="text-xs font-semibold dark:text-gray-200 text-gray-800">Groups</h2>
        <button
          onClick={onCreateClick}
          className="px-2.5 py-1 text-xs rounded-lg bg-blue-600 text-white hover:bg-blue-700 font-medium"
        >
          + New Group
        </button>
      </div>
      {groups.length === 0 ? (
        <p className="text-xs dark:text-gray-500 text-gray-400">No groups yet.</p>
      ) : (
        <table className="w-full text-xs">
          <thead>
            <tr className="border-b dark:border-[#2D3748] border-gray-200">
              <th className="text-left py-2 font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider text-[10px]">Name</th>
              <th className="text-left py-2 font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider text-[10px]">Description</th>
              <th className="py-2" />
            </tr>
          </thead>
          <tbody>
            {groups.map(group => (
              <tr key={group.id} className="border-t dark:border-[#2D3748]/60 border-gray-100">
                <td className="py-2 pr-2">
                  <button
                    onClick={() => onSelect(group)}
                    className="text-blue-400 hover:text-blue-300 underline underline-offset-2"
                  >
                    {group.name}
                  </button>
                </td>
                <td className="py-2 pr-2 dark:text-gray-400 text-gray-500">{group.description}</td>
                <td className="py-2 text-right">
                  <button onClick={() => onDelete(group.id)} className="text-red-400 hover:text-red-300">
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

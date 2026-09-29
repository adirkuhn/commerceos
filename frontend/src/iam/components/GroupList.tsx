import { useTranslation } from 'react-i18next'
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
  const { t } = useTranslation(['iam', 'common'])
  if (loading) return <p className="text-xs text-ink-3">{t('groups.loading')}</p>
  if (error) return <p className="text-xs text-danger">{t('errorPrefix', { error })}</p>

  return (
    <div>
      <div className="flex items-center justify-between mb-3">
        <h2 className="text-xs font-semibold text-ink">{t('groups.title')}</h2>
        <button
          onClick={onCreateClick}
          className="px-2.5 py-1 text-xs rounded-lg bg-accent-solid text-white hover:bg-accent-solid-hover font-medium"
        >
          {t('groups.newButton')}
        </button>
      </div>
      {groups.length === 0 ? (
        <p className="text-xs text-ink-4">{t('groups.none')}</p>
      ) : (
        <table className="w-full text-xs">
          <thead>
            <tr className="border-b border-line">
              <th className="text-left py-2 font-medium text-ink-4 uppercase tracking-wider text-2xs">{t('common:fields.name')}</th>
              <th className="text-left py-2 font-medium text-ink-4 uppercase tracking-wider text-2xs">{t('common:fields.description')}</th>
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
                    {t('common:actions.delete')}
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

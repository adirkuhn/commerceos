import { GripVertical, Pencil, Trash2 } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import type { AIConnection } from '../integrations/types'
import { connectionStatusLabel } from '../integrations/connectionStatus'
import { STATUS_STYLES } from './IntegrationsSettingsSection'
import { connectionName } from './aiProviders'

// One connection in the ordered list: drag handle (pointer, touch or
// keyboard: Space, arrows, Space), its place, name, model, status and
// actions.
export function AIConnectionRow({ connection, place, onToggle, onEdit, onDelete }: {
  connection: AIConnection
  place: number
  onToggle: (enabled: boolean) => void
  onEdit: () => void
  onDelete: () => void
}) {
  const { t } = useTranslation(['settings', 'common'])
  const { attributes, listeners, setNodeRef, setActivatorNodeRef, transform, transition, isDragging } = useSortable({ id: connection.id })
  const name = connectionName(t, connection)

  return (
    <li ref={setNodeRef} data-ai-row
      style={{ transform: CSS.Transform.toString(transform), transition }}
      className={clsx('flex items-center gap-2 rounded-lg border border-line bg-surface px-2 py-2',
        isDragging && 'relative z-10 shadow-lg', !connection.enabled && 'opacity-60')}>
      <button ref={setActivatorNodeRef} type="button" {...attributes} {...listeners}
        aria-label={t('ai.dragHandle', { name })}
        className="p-1 rounded text-ink-4 hover:text-ink-2 cursor-grab active:cursor-grabbing touch-none focus:outline-none focus:ring-1 focus:ring-focus">
        <GripVertical size={14} />
      </button>
      <span className="w-5 text-2xs text-ink-4 tabular-nums text-right">{place}.</span>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <p className="text-xs font-medium text-ink truncate">{name}</p>
          <span className={clsx('px-1.5 py-0.5 rounded-full text-2xs font-medium capitalize', STATUS_STYLES[connection.status])}>
            {connectionStatusLabel(connection.status)}
          </span>
        </div>
        <p className="text-2xs text-ink-4 truncate">{connection.model || t('ai.providerDefault')}</p>
      </div>
      <label className="flex items-center gap-1.5 text-2xs text-ink-3 cursor-pointer">
        <input type="checkbox" role="switch" checked={connection.enabled}
          aria-label={t('ai.enabledFor', { name })}
          onChange={e => onToggle(e.target.checked)} className="rounded" />
        {t('ai.enabled')}
      </label>
      <button type="button" onClick={onEdit} aria-label={t('ai.editFor', { name })}
        className="p-1.5 rounded-lg text-ink-3 hover:bg-hover"><Pencil size={13} /></button>
      <button type="button" onClick={onDelete} aria-label={t('ai.deleteFor', { name })}
        className="p-1.5 rounded-lg text-ink-3 hover:text-danger hover:bg-hover"><Trash2 size={13} /></button>
    </li>
  )
}

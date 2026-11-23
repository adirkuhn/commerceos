import { useState } from 'react'
import { Bot, Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import {
  DndContext, KeyboardSensor, PointerSensor, TouchSensor, closestCenter, useSensor, useSensors, type DragEndEvent,
} from '@dnd-kit/core'
import { SortableContext, arrayMove, sortableKeyboardCoordinates, verticalListSortingStrategy } from '@dnd-kit/sortable'
import { useAuth } from '../auth/AuthContext'
import { useAIConnections } from '../integrations/hooks/useAIConnections'
import type { AIConnection } from '../integrations/types'
import type { AIConnectionPatch } from '../integrations/api'
import { useConfirm } from '../components/common/useConfirm'
import { apiErrorText } from '../api/apiErrorText'
import { AIConnectionForm, type AIConnectionFormValues } from './AIConnectionForm'
import { AIConnectionRow } from './AIConnectionRow'
import { connectionName } from './aiProviders'

/** The fields an edit changed; the key only when a new one was typed. */
export function changedFields(initial: AIConnection, v: AIConnectionFormValues): AIConnectionPatch {
  const patch: AIConnectionPatch = {}
  if (v.provider !== initial.provider) patch.provider = v.provider
  if (v.apiKey) patch.apiKey = v.apiKey
  if (v.baseUrl !== (initial.base_url ?? '')) patch.baseUrl = v.baseUrl
  if (v.model !== (initial.model ?? '')) patch.model = v.model
  if (v.fallbackModels !== (initial.fallback_models ?? '')) patch.fallbackModels = v.fallbackModels
  if (v.label !== initial.label) patch.label = v.label
  return patch
}

// The tenant's AI connections, in the order every AI feature tries them
// (like a router: the first enabled one, then the next on any failure).
// Dragging a row saves the new order right away.
export function AIIntegrationCard() {
  const { confirm, confirmDialog } = useConfirm()
  const { t } = useTranslation(['settings', 'common'])
  const { token } = useAuth()
  const { connections, loading, error, create, update, remove, reorder } = useAIConnections(token ?? '')
  const [editing, setEditing] = useState<AIConnection | 'new' | null>(null)
  const [actionError, setActionError] = useState('')
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 150, tolerance: 5 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  )

  const run = async (action: () => Promise<unknown>) => {
    setActionError('')
    try {
      await action()
    } catch (err) {
      setActionError(apiErrorText(err) ?? t('ai.saveFailed'))
    }
  }

  const handleDragEnd = ({ active, over }: DragEndEvent) => {
    if (!over || active.id === over.id) return
    const ids = connections.map(c => c.id)
    void reorder(arrayMove(ids, ids.indexOf(String(active.id)), ids.indexOf(String(over.id))))
  }

  const handleSave = async (values: AIConnectionFormValues) => {
    if (editing === 'new') {
      await create({ ...values })
    } else if (editing) {
      await update(editing.id, changedFields(editing, values))
    }
    setEditing(null)
  }

  const handleDelete = async (c: AIConnection) => {
    if (!(await confirm({
      title: t('ai.deleteTitle', { name: connectionName(t, c) }), message: t('ai.deleteMessage'),
      confirmLabel: t('ai.delete'), destructive: true,
    }))) return
    await run(() => remove(c.id))
  }

  return (
    <div className="bg-raised rounded-xl border border-line p-4">
      <div className="flex items-center gap-3">
        <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-blue-500 to-cyan-600 flex items-center justify-center text-white shrink-0">
          <Bot size={14} />
        </div>
        <div className="min-w-0">
          <p className="text-xs font-medium text-ink">{t('ai.title')}</p>
          <p className="text-2xs text-ink-4 mt-0.5">
            {loading ? t('common:status.loading') : connections.length ? t('ai.routingHelp') : t('ai.notConfigured')}
          </p>
        </div>
      </div>

      {connections.length > 0 && (
        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
          <SortableContext items={connections.map(c => c.id)} strategy={verticalListSortingStrategy}>
            <ol className="mt-3 space-y-1.5" aria-label={t('ai.listLabel')}>
              {connections.map((c, i) => (
                <AIConnectionRow key={c.id} connection={c} place={i + 1}
                  onToggle={enabled => run(() => update(c.id, { enabled }))}
                  onEdit={() => setEditing(c)}
                  onDelete={() => handleDelete(c)} />
              ))}
            </ol>
          </SortableContext>
        </DndContext>
      )}

      <button type="button" onClick={() => setEditing('new')}
        className="mt-3 inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-fill text-ink-2 text-xs font-medium hover:bg-hover transition-colors">
        <Plus size={13} /> {t('ai.add')}
      </button>
      {(error || actionError) && <p className="mt-2 text-2xs text-danger">{actionError || error}</p>}

      {editing && (
        <AIConnectionForm initial={editing === 'new' ? null : editing} onSave={handleSave} onClose={() => setEditing(null)} />
      )}
      {confirmDialog}
    </div>
  )
}

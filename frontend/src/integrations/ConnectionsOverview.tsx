import { Plug, Webhook, ChevronRight } from 'lucide-react'
import type { Connection } from './types'
import type { ConnectionPanel } from './hooks/useIntegrationsRoute'
import type { TestResult } from './hooks/useTestConnection'
import { ConnectionCard } from './ConnectionCard'

// Shortcut card to the Webhooks admin page (owners/admins only).
export function WebhooksShortcut({ onOpen }: { onOpen: () => void }) {
  return (
    <button
      onClick={onOpen}
      className="flex items-center justify-between px-4 py-3 rounded-xl bg-raised border border-line hover:border-accent transition-colors text-left"
    >
      <div className="flex items-center gap-3">
        <div className="w-8 h-8 rounded-lg bg-accent/10 flex items-center justify-center shrink-0">
          <Webhook size={15} className="text-accent" />
        </div>
        <div>
          <p className="text-xs font-semibold text-ink">Webhooks</p>
          <p className="text-2xs text-ink-4">Fan-out rules and every active provider subscription, in one place</p>
        </div>
      </div>
      <ChevronRight size={16} className="text-ink-4 shrink-0" />
    </button>
  )
}

interface Props {
  connections: Connection[]
  loading: boolean
  error: string | null
  testingId: string | null
  testResult: TestResult | null
  onBrowse: () => void
  onOpenPanel: (conn: Connection, panel: ConnectionPanel) => void
  onTest: (id: string) => void
  onDisconnect: (id: string) => void
}

// The connected-integrations grid, or the empty state when there are none.
export function ConnectionsOverview({ connections, loading, error, testingId, testResult, onBrowse, onOpenPanel, onTest, onDisconnect }: Props) {
  if (connections.length === 0 && !loading) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center text-center">
        <div className="w-12 h-12 rounded-xl bg-raised border border-line flex items-center justify-center mb-3">
          <Plug size={20} className="text-ink-4" />
        </div>
        <p className="text-sm font-medium text-ink-2 mb-1">No integrations connected</p>
        <p className="text-xs text-ink-4 mb-4">Connect a store to start syncing orders and inventory</p>
        <button
          onClick={onBrowse}
          className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover transition-colors"
        >
          Browse integrations
        </button>
      </div>
    )
  }

  return (
    <>
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-ink">Connected integrations</h2>
        <button
          onClick={onBrowse}
          className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover transition-colors"
        >
          Browse integrations
        </button>
      </div>
      {error && <p className="text-xs text-danger">{error}</p>}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
        {connections.map(conn => (
          <ConnectionCard
            key={conn.id}
            connection={conn}
            testing={testingId === conn.id}
            testResult={testResult && testResult.id === conn.id ? testResult : null}
            onOpenPanel={panel => onOpenPanel(conn, panel)}
            onTest={() => onTest(conn.id)}
            onDisconnect={() => onDisconnect(conn.id)}
          />
        ))}
      </div>
    </>
  )
}

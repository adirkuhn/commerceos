import type { ShippingConnection, ShippingConnections } from '../shipping/types'

// Test fixture: a GET /shipping/connections body for these connections,
// defaulting to the first (the backend's "oldest" rule) unless one is
// given.
export function shippingConnectionsOf(
  connections: ReadonlyArray<Pick<ShippingConnection, 'id' | 'provider'> & Partial<ShippingConnection>>,
  defaultId: string | null = connections[0]?.id ?? null,
  configuredId: string | null = null,
): ShippingConnections {
  return {
    connections: connections.map(c => ({ status: 'active', created_at: '2026-01-01T00:00:00Z', ...c, is_default: c.id === defaultId })),
    default_connection_id: defaultId,
    configured_connection_id: configuredId,
  }
}

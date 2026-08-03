// Mirrors pms/productsBus.ts. No polling, no push (SSE/WebSocket) — per
// design, the bell fetches fresh on mount, when opened, and right after an
// action known to have created a notification server-side (e.g. enqueueing
// an import — the "started" notification is written synchronously in that
// same request before the 202 comes back).
type Listener = () => void

const listeners = new Set<Listener>()

export function notifyNotificationsChanged(): void {
  listeners.forEach(l => l())
}

export function subscribeNotificationsChanged(listener: Listener): () => void {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

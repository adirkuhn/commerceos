import { wait } from '@/lib/utils'

const LATENCY_MS = Number(import.meta.env.VITE_MOCK_LATENCY_MS ?? 350)

// simulateLatency keeps loading/skeleton states visibly real in dev
// instead of resolving instantly -- every mock repository method awaits
// this before returning.
export function simulateLatency(): Promise<void> {
  if (import.meta.env.MODE === 'test') return Promise.resolve()
  return wait(LATENCY_MS)
}

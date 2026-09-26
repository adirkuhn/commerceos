import { createContext, useContext, useEffect, useMemo, useRef, type ReactNode } from 'react'

// Stock invalidation: a stock action on one SKU can change another SKU's
// stock too (a case split adds units to the case's unit SKU, a portion
// draws from the bulk SKU), and that other SKU may be showing on screen
// in a different product row. The acting view names every SKU it touched
// via useInvalidateStock; each view showing a SKU's stock subscribes with
// useOnStockInvalidated and reloads itself. Outside a provider both are
// no-ops, so hooks using them still work on screens that don't need it.

type Listener = { skuIds: ReadonlySet<string>; onInvalidate: () => void }

interface StockInvalidationBus {
  invalidate: (skuIds: string[]) => void
  subscribe: (listener: Listener) => () => void
}

function createStockInvalidationBus(): StockInvalidationBus {
  const listeners = new Set<Listener>()
  return {
    invalidate: skuIds => {
      for (const l of [...listeners]) {
        if (skuIds.some(id => l.skuIds.has(id))) l.onInvalidate()
      }
    },
    subscribe: listener => {
      listeners.add(listener)
      return () => { listeners.delete(listener) }
    },
  }
}

const noopBus: StockInvalidationBus = { invalidate: () => {}, subscribe: () => () => {} }

const StockInvalidationContext = createContext<StockInvalidationBus>(noopBus)

export function StockInvalidationProvider({ children }: { children: ReactNode }) {
  const bus = useMemo(createStockInvalidationBus, [])
  return <StockInvalidationContext.Provider value={bus}>{children}</StockInvalidationContext.Provider>
}

export function useInvalidateStock(): (skuIds: string[]) => void {
  return useContext(StockInvalidationContext).invalidate
}

// SKU ids are UUIDs (never contain a comma), so the joined key is a
// stable effect dependency for a list that's a new array every render.
export function useOnStockInvalidated(skuIds: string[], onInvalidate: () => void): void {
  const bus = useContext(StockInvalidationContext)
  const callback = useRef(onInvalidate)
  useEffect(() => { callback.current = onInvalidate })
  const key = skuIds.join(',')
  useEffect(() => {
    if (!key) return
    return bus.subscribe({ skuIds: new Set(key.split(',')), onInvalidate: () => callback.current() })
  }, [bus, key])
}

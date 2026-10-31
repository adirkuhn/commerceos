import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { emptyPackLineHistory, orderPackLines, packLineChanges, type PackLine, type PackLineHistory } from './packLineOrder'

// How long a scan's highlight lasts; a line finished from partly packed
// stays in place that long before it moves to the end of the list.
export const PACK_LINE_SETTLE_MS = 700

export type PackLineHighlight = 'done' | 'partial'

// The highlight on screen: a pop with a green (done) or amber (partly
// packed) ring. The line on top keeps its green ring until the next scan.
export const PACK_LINE_HIGHLIGHT_CLASS: Record<PackLineHighlight, string> = {
  done: 'animate-pack-flash ring-2 ring-green-500',
  partial: 'animate-pack-flash ring-2 ring-amber-400',
}

// The last list state this hook reacted to. A change only animates when
// it happens within the same, already loaded order.
interface Seen {
  sessionKey: string
  loaded: boolean
  packed: Map<string, number>
}

// The packing list in working order (see orderPackLines), reacting to
// scans:
//   - a line finished in one scan goes on top with a green ring that stays
//     until the next scan, which sends it to the done lines at the end;
//   - a line a scan partly packs joins the partly packed lines (in first-
//     scanned order) with an amber pulse;
//   - a line finished from partly packed flashes green where it is, then
//     moves to the done lines at the end.
// Opening an order (or switching to another) just sorts, without effects.
export function usePackLineOrder<L extends PackLine>(sessionKey: string, lines: L[], packed: Map<string, number>) {
  const [history, setHistory] = useState<PackLineHistory>(emptyPackLineHistory)
  // Short-lived rings (a scan's pulse); the top line's lasting ring is added
  // from history when returned.
  const [fadingHighlight, setHighlight] = useState<Map<string, PackLineHighlight>>(() => new Map())
  const seen = useRef<Seen | null>(null)
  const timers = useRef(new Set<ReturnType<typeof setTimeout>>())

  // packed is a new Map on every render; react to its contents only.
  const signature = lines.map(l => `${l.id}:${l.quantity}:${packed.get(l.id) ?? 0}`).join('|')

  useLayoutEffect(() => {
    const before = seen.current
    seen.current = { sessionKey, loaded: lines.length > 0, packed: new Map(packed) }
    if (!before || before.sessionKey !== sessionKey || !before.loaded) {
      setHistory(emptyPackLineHistory())
      setHighlight(new Map())
      return
    }
    const { completed, advanced } = packLineChanges(lines, before.packed, packed)
    if (completed.size === 0 && advanced.length === 0) return

    // Finished in one scan -> the new top; from partly packed -> settles.
    const finishedAtOnce = [...completed].filter(([, was]) => was === 'todo').map(([id]) => id)
    const settled = [...completed].filter(([, was]) => was === 'partial').map(([id]) => id)
    const top = finishedAtOnce[finishedAtOnce.length - 1]

    setHistory(prev => ({
      top,
      partialOrder: [...prev.partialOrder, ...advanced.filter(id => !prev.partialOrder.includes(id))],
      settling: new Map([...prev.settling, ...settled.map(id => [id, 'partial'] as const)]),
    }))
    // The top line's ring comes from history (below); these fade.
    const fading = [...settled, ...advanced]
    if (fading.length === 0) return
    setHighlight(prev => new Map([
      ...prev,
      ...settled.map(id => [id, 'done'] as const),
      ...advanced.map(id => [id, 'partial'] as const),
    ]))
    const timer = setTimeout(() => {
      timers.current.delete(timer)
      setHistory(prev => ({ ...prev, settling: without(prev.settling, settled) }))
      setHighlight(prev => without(prev, fading))
    }, PACK_LINE_SETTLE_MS)
    timers.current.add(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- signature stands for lines + packed
  }, [sessionKey, signature])

  useEffect(() => {
    const pending = timers.current
    return () => pending.forEach(clearTimeout)
  }, [])

  const ordered = useMemo(
    () => orderPackLines(lines, packed, history),
    // eslint-disable-next-line react-hooks/exhaustive-deps -- signature stands for lines + packed
    [signature, history],
  )
  const highlight = useMemo(() => {
    const top = lines.find(l => l.id === history.top)
    if (!top || (packed.get(top.id) ?? 0) < top.quantity) return fadingHighlight
    return new Map([...fadingHighlight, [top.id, 'done' as const]])
    // eslint-disable-next-line react-hooks/exhaustive-deps -- signature stands for packed
  }, [fadingHighlight, history.top, signature])
  return { lines: ordered, highlight }
}

function without<V>(map: Map<string, V>, ids: string[]): Map<string, V> {
  if (!ids.some(id => map.has(id))) return map
  const next = new Map(map)
  ids.forEach(id => next.delete(id))
  return next
}

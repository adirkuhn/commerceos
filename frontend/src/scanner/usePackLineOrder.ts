import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { orderPackLines, packLineChanges, type PackLine, type PackLineState } from './packLineOrder'

// How long a just-scanned line shows its highlight; a completed line stays
// in place that long before it moves to the end of the list.
export const PACK_LINE_SETTLE_MS = 700

export type PackLineHighlight = 'done' | 'partial'

// The highlight on screen: a short pop with a green (done) or amber
// (partly packed) ring.
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
// scans: a line a scan completes flashes where it is, then moves to the
// end; a line a scan only partly packs moves to the top with a pulse.
// Opening an order (or switching to another) just sorts, without effects.
export function usePackLineOrder<L extends PackLine>(sessionKey: string, lines: L[], packed: Map<string, number>) {
  const [settling, setSettling] = useState<Map<string, PackLineState>>(() => new Map())
  const [highlight, setHighlight] = useState<Map<string, PackLineHighlight>>(() => new Map())
  const seen = useRef<Seen | null>(null)
  const timers = useRef(new Set<ReturnType<typeof setTimeout>>())

  // packed is a new Map on every render; react to its contents only.
  const signature = lines.map(l => `${l.id}:${l.quantity}:${packed.get(l.id) ?? 0}`).join('|')

  useLayoutEffect(() => {
    const before = seen.current
    seen.current = { sessionKey, loaded: lines.length > 0, packed: new Map(packed) }
    if (!before || before.sessionKey !== sessionKey || !before.loaded) {
      setSettling(new Map())
      setHighlight(new Map())
      return
    }
    const { completed, advanced } = packLineChanges(lines, before.packed, packed)
    if (completed.size === 0 && advanced.length === 0) return

    const touched = [...completed.keys(), ...advanced]
    setSettling(prev => new Map([...prev, ...completed]))
    setHighlight(prev => new Map([
      ...prev,
      ...[...completed.keys()].map(id => [id, 'done'] as const),
      ...advanced.map(id => [id, 'partial'] as const),
    ]))
    const timer = setTimeout(() => {
      timers.current.delete(timer)
      setSettling(prev => without(prev, touched))
      setHighlight(prev => without(prev, touched))
    }, PACK_LINE_SETTLE_MS)
    timers.current.add(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- signature stands for lines + packed
  }, [sessionKey, signature])

  useEffect(() => {
    const pending = timers.current
    return () => pending.forEach(clearTimeout)
  }, [])

  const ordered = useMemo(
    () => orderPackLines(lines, packed, settling),
    // eslint-disable-next-line react-hooks/exhaustive-deps -- signature stands for lines + packed
    [signature, settling],
  )
  return { lines: ordered, highlight }
}

function without<V>(map: Map<string, V>, ids: string[]): Map<string, V> {
  if (!ids.some(id => map.has(id))) return map
  const next = new Map(map)
  ids.forEach(id => next.delete(id))
  return next
}

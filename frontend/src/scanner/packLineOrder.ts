// Packing list order, built so the packer always sees what they just did:
//   1. the line just finished in ONE scan (it was not packed at all before),
//      until the next scan of any kind;
//   2. partly packed lines, in the order they were first scanned;
//   3. lines not packed yet, in the order's own order;
//   4. done lines, in the order's own order -- including a line finished
//      from partly packed, which goes straight here.
// A fresh order therefore opens exactly as the order lists it.

export interface PackLine {
  id: string
  quantity: number
}

export type PackLineState = 'partial' | 'todo' | 'done'

// What the scans so far tell the ordering, beyond packed quantities.
export interface PackLineHistory {
  // The line the last scan finished from not packed at all.
  top?: string
  // Partly packed lines, in the order they were first scanned.
  partialOrder: string[]
  // Lines that just completed, with the state they had before: they keep
  // that place while their completion flashes (see usePackLineOrder).
  settling: Map<string, PackLineState>
}

export const emptyPackLineHistory = (): PackLineHistory => ({ partialOrder: [], settling: new Map() })

const GROUP_RANK = { top: 0, partial: 1, todo: 2, done: 3 } as const

export function packLineState(line: PackLine, packed: number): PackLineState {
  if (packed >= line.quantity) return 'done'
  return packed > 0 ? 'partial' : 'todo'
}

export function orderPackLines<L extends PackLine>(
  lines: L[], packed: Map<string, number>, history: PackLineHistory = emptyPackLineHistory(),
): L[] {
  const firstScanned = new Map(history.partialOrder.map((id, i) => [id, i]))
  const placed = lines.map((line, index) => {
    const state = history.settling.get(line.id) ?? packLineState(line, packed.get(line.id) ?? 0)
    const group = line.id === history.top && state === 'done' ? 'top' : state
    // Partly packed lines go by first scan; ones from before this screen
    // was open (no scan seen) come after those, in the order's own order.
    const within = group === 'partial' ? (firstScanned.get(line.id) ?? lines.length + index) : index
    return { line, rank: GROUP_RANK[group], within }
  })
  return placed
    .sort((x, y) => x.rank - y.rank || x.within - y.within)
    .map(x => x.line)
}

export interface PackLineChanges {
  // Lines a scan just finished, with the state they were in before.
  completed: Map<string, PackLineState>
  // Lines a scan just packed more of without finishing them.
  advanced: string[]
}

export function packLineChanges(lines: PackLine[], before: Map<string, number>, after: Map<string, number>): PackLineChanges {
  const completed = new Map<string, PackLineState>()
  const advanced: string[] = []
  for (const line of lines) {
    const was = before.get(line.id) ?? 0
    const now = after.get(line.id) ?? 0
    if (now <= was) continue
    const wasState = packLineState(line, was)
    if (packLineState(line, now) === 'done') {
      if (wasState !== 'done') completed.set(line.id, wasState)
    } else {
      advanced.push(line.id)
    }
  }
  return { completed, advanced }
}

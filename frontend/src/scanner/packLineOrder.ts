// Packing list order: lines being worked on stay in reach, finished ones
// get out of the way. Partly packed lines go on top, then lines not packed
// yet, then done lines at the end; each group keeps the order's own line
// order, so a fresh order opens exactly as the order lists it.

export interface PackLine {
  id: string
  quantity: number
}

export type PackLineState = 'partial' | 'todo' | 'done'

const GROUP_RANK: Record<PackLineState, number> = { partial: 0, todo: 1, done: 2 }

export function packLineState(line: PackLine, packed: number): PackLineState {
  if (packed >= line.quantity) return 'done'
  return packed > 0 ? 'partial' : 'todo'
}

// settling: lines that just completed, with the state they had before --
// they keep that place while their completion flashes (see usePackLineOrder).
export function orderPackLines<L extends PackLine>(
  lines: L[], packed: Map<string, number>, settling: Map<string, PackLineState> = new Map(),
): L[] {
  const rank = (l: L) => GROUP_RANK[settling.get(l.id) ?? packLineState(l, packed.get(l.id) ?? 0)]
  return lines
    .map((line, index) => ({ line, index, rank: rank(line) }))
    .sort((x, y) => x.rank - y.rank || x.index - y.index)
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

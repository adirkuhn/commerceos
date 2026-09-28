/** Swaps item i with its neighbour d steps away (±1); out of range is a no-op. */
export function moveItem<T>(list: readonly T[], i: number, d: number): T[] {
  const j = i + d
  if (i < 0 || j < 0 || i >= list.length || j >= list.length) return [...list]
  const next = [...list]
  ;[next[i], next[j]] = [next[j], next[i]]
  return next
}

export function replaceItem<T>(list: readonly T[], i: number, item: T): T[] {
  return list.map((v, k) => (k === i ? item : v))
}

export function removeItem<T>(list: readonly T[], i: number): T[] {
  return list.filter((_, k) => k !== i)
}

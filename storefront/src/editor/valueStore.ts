/** One observable value, for state the canvas's separate React roots read
 *  with useSyncExternalStore. */
export class ValueStore<T> {
  private value: T
  private readonly listeners = new Set<() => void>()

  constructor(initial: T) {
    this.value = initial
  }

  snapshot = (): T => this.value

  subscribe = (fn: () => void): (() => void) => {
    this.listeners.add(fn)
    return () => this.listeners.delete(fn)
  }

  set(value: T) {
    if (Object.is(value, this.value)) return
    this.value = value
    this.listeners.forEach(l => l())
  }
}

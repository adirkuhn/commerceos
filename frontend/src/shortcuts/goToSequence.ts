// "g then <letter>" jump-to-page sequences (g p = Products). Pure state
// machine so the timing rules are testable without a DOM.
export const GO_TO_TIMEOUT_MS = 1500

export interface GoToSequence<T> {
  // Feed one key press; returns the target when a sequence completes.
  press(key: string, now: number): T | null
}

export function createGoToSequence<T>(targets: Readonly<Record<string, T>>): GoToSequence<T> {
  let armedAt: number | null = null
  return {
    press(rawKey, now) {
      const key = rawKey.toLowerCase()
      const armed = armedAt !== null && now - armedAt <= GO_TO_TIMEOUT_MS
      armedAt = null
      if (armed && key in targets) return targets[key]
      if (!armed && key === 'g') armedAt = now
      return null
    },
  }
}

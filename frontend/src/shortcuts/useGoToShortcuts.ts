import { useEffect, useRef } from 'react'
import { createGoToSequence } from './goToSequence'

function isEditable(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false
  if (target.isContentEditable || target.getAttribute('contenteditable') === 'true') return true
  return ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName)
}

// Listens for "g then <letter>" anywhere in the app, except while typing
// in a field or holding a modifier (those keys belong to the field or to
// browser/system shortcuts).
export function useGoToShortcuts<T>(targets: Readonly<Record<string, T>>, onGo: (target: T) => void): void {
  const onGoRef = useRef(onGo)
  onGoRef.current = onGo

  useEffect(() => {
    const sequence = createGoToSequence(targets)
    const handler = (e: KeyboardEvent) => {
      if (e.metaKey || e.ctrlKey || e.altKey || isEditable(e.target)) return
      const hit = sequence.press(e.key, e.timeStamp || Date.now())
      if (hit !== null) {
        e.preventDefault()
        onGoRef.current(hit)
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [targets])
}

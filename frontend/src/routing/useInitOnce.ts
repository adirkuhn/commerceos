import { useEffect, useRef, useState } from 'react'

// Runs `init` once per `key` as soon as `ready` -- for a form whose open
// state comes from the URL (e.g. "/settings/tax/classes/<id>"): on a
// refresh the record it edits arrives after the first render, so the form
// can't be filled at click time anymore. Returns whether the current key
// has been initialised, so the caller can hold the form back until then
// instead of flashing empty fields. key === null means "closed"; reopening
// the same key later initialises again.
export function useInitOnce(key: string | null, ready: boolean, init: () => void): boolean {
  const [doneFor, setDoneFor] = useState<string | null>(null)
  const initRef = useRef(init)
  initRef.current = init

  useEffect(() => {
    if (key === null) {
      setDoneFor(null)
      return
    }
    if (doneFor === key || !ready) return
    initRef.current()
    setDoneFor(key)
  }, [key, ready, doneFor])

  return key !== null && doneFor === key
}

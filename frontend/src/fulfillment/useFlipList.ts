import { useCallback, useLayoutEffect, useRef } from 'react'

const SLIDE = 'transform 300ms ease-in-out'

// Rows that change place in a list slide to their new place instead of
// jumping (FLIP: measure before, measure after, animate the difference).
// order is the rows' keys in render order; ref(key) goes on each row.
// Reduce Motion stops the slide (index.css shortens every transition).
export function useFlipList(order: string[]) {
  const nodes = useRef(new Map<string, HTMLElement>())
  const tops = useRef(new Map<string, number>())
  const orderKey = order.join('|')

  useLayoutEffect(() => {
    const previous = tops.current
    const next = new Map<string, number>()
    nodes.current.forEach((el, key) => {
      const top = el.getBoundingClientRect().top
      next.set(key, top)
      const before = previous.get(key)
      if (before === undefined || before === top) return
      el.style.transition = 'none'
      el.style.transform = `translateY(${before - top}px)`
      void el.offsetHeight // apply the start position before animating
      el.style.transition = SLIDE
      el.style.transform = ''
    })
    tops.current = next
  }, [orderKey])

  return useCallback((key: string) => (el: HTMLElement | null) => {
    if (el) nodes.current.set(key, el)
    else nodes.current.delete(key)
  }, [])
}

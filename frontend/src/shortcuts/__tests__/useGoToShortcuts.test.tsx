// @vitest-environment jsdom
import { fireEvent, render } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { useGoToShortcuts } from '../useGoToShortcuts'

function Harness({ onGo }: { onGo: (nav: string) => void }) {
  useGoToShortcuts({ d: 'dashboard', p: 'products' }, onGo)
  return (
    <>
      <input aria-label="Name" />
      <div contentEditable aria-label="Notes" />
    </>
  )
}

function press(key: string, target: Element = document.body, init: KeyboardEventInit = {}) {
  fireEvent.keyDown(target, { key, ...init })
}

describe('useGoToShortcuts', () => {
  it('navigates on g then a target letter', () => {
    const onGo = vi.fn()
    render(<Harness onGo={onGo} />)
    press('g')
    press('p')
    expect(onGo).toHaveBeenCalledWith('products')
  })

  it('ignores keys typed into a text field', () => {
    const onGo = vi.fn()
    const { getByLabelText } = render(<Harness onGo={onGo} />)
    const input = getByLabelText('Name')
    press('g', input)
    press('p', input)
    expect(onGo).not.toHaveBeenCalled()
  })

  it('ignores keys typed into editable content', () => {
    const onGo = vi.fn()
    const { getByLabelText } = render(<Harness onGo={onGo} />)
    const notes = getByLabelText('Notes')
    press('g', notes)
    press('d', notes)
    expect(onGo).not.toHaveBeenCalled()
  })

  it('ignores chords with modifier keys (browser/system shortcuts)', () => {
    const onGo = vi.fn()
    render(<Harness onGo={onGo} />)
    press('g', document.body, { metaKey: true })
    press('p')
    press('g')
    press('p', document.body, { ctrlKey: true })
    expect(onGo).not.toHaveBeenCalled()
  })
})

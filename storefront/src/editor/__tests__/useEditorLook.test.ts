// @vitest-environment jsdom
import { renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { DEFAULT_EDITOR_APPEARANCE } from '../editorLook'
import { useEditorLook } from '../useEditorLook'

let systemDark = false
const listeners: Array<() => void> = []
beforeEach(() => {
  systemDark = false
  listeners.length = 0
  document.documentElement.className = ''
  document.head.innerHTML = ''
  vi.stubGlobal('matchMedia', () => ({
    get matches() { return systemDark },
    addEventListener: (_: string, fn: () => void) => listeners.push(fn),
    removeEventListener: vi.fn(),
  }))
})

describe('useEditorLook', () => {
  it('goes dark when the admin was dark', () => {
    renderHook(() => useEditorLook('dark', undefined))
    expect(document.documentElement).toHaveClass('dark')
  })

  it('follows the OS live when the admin follows the system', () => {
    renderHook(() => useEditorLook('system', undefined))
    expect(document.documentElement).not.toHaveClass('dark')

    systemDark = true
    listeners.forEach(fn => fn())

    expect(document.documentElement).toHaveClass('dark')
  })

  it('loads the shop font and applies its accent', () => {
    renderHook(() => useEditorLook('light', { ...DEFAULT_EDITOR_APPEARANCE, font: 'inter', accent_color: '#0B6E4F' }))

    expect(document.getElementById('editor-font')?.getAttribute('href')).toContain('family=Inter')
    expect(document.getElementById('editor-appearance')?.textContent).toContain('--ed-accent: 11 110 79;')
  })

  it('loads the default font and adds no overrides for the default appearance', () => {
    renderHook(() => useEditorLook('light', DEFAULT_EDITOR_APPEARANCE))

    expect(document.getElementById('editor-font')?.getAttribute('href')).toContain('Atkinson')
    expect(document.getElementById('editor-appearance')).toBeNull()
  })
})

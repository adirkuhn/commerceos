// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { createRef } from 'react'
import { useScanInput } from '../useScanInput'

vi.mock('../errorSound', () => ({ playErrorSound: vi.fn() }))
import { playErrorSound } from '../errorSound'

describe('useScanInput', () => {
  it('a scan error beeps and blocks until dismissed, then refocuses the input', () => {
    const ref = createRef<HTMLInputElement>()
    const input = document.createElement('input')
    document.body.appendChild(input)
    ;(ref as { current: HTMLInputElement | null }).current = input
    const { result } = renderHook(() => useScanInput(ref))

    act(() => result.current.showScanError('Item not on this order'))
    expect(playErrorSound).toHaveBeenCalled()
    expect(result.current.scanErrorAlert).toBe('Item not on this order')

    act(() => result.current.dismissScanError())
    expect(result.current.scanErrorAlert).toBeNull()
    expect(document.activeElement).toBe(input)
  })
})

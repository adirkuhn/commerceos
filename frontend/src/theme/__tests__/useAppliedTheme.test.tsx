// @vitest-environment jsdom
import { act, renderHook } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { useAppliedTheme } from '../useAppliedTheme'

afterEach(() => document.documentElement.classList.remove('dark'))

describe('useAppliedTheme', () => {
  it('reads the appearance applied to the document', () => {
    document.documentElement.classList.add('dark')
    const { result } = renderHook(() => useAppliedTheme())
    expect(result.current).toBe('dark')
  })

  it('updates when the appearance changes, without needing app context', async () => {
    const { result } = renderHook(() => useAppliedTheme())
    expect(result.current).toBe('light')

    await act(async () => { document.documentElement.classList.add('dark') })

    expect(result.current).toBe('dark')
  })
})

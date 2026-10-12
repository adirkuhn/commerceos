// @vitest-environment jsdom
import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { DESIGN_CLASS, DESIGN_STORAGE_KEY } from '../designPreference'
import { applyDesign, initDesign, useDesign } from '../useDesign'

beforeEach(() => {
  localStorage.clear()
  document.documentElement.classList.remove(DESIGN_CLASS)
})

afterEach(() => {
  localStorage.clear()
  document.documentElement.classList.remove(DESIGN_CLASS)
})

describe('useDesign', () => {
  it('reads the design applied to the document', () => {
    applyDesign('depot')
    const { result } = renderHook(() => useDesign())
    expect(result.current.design).toBe('depot')
  })

  it('reads classic when no design class is applied', () => {
    const { result } = renderHook(() => useDesign())
    expect(result.current.design).toBe('classic')
  })

  it('switches the document design and remembers the choice', async () => {
    const { result } = renderHook(() => useDesign())

    await act(async () => { result.current.setDesign('depot') })

    expect(result.current.design).toBe('depot')
    expect(document.documentElement.classList.contains(DESIGN_CLASS)).toBe(true)
    expect(localStorage.getItem(DESIGN_STORAGE_KEY)).toBe('depot')

    await act(async () => { result.current.setDesign('classic') })

    expect(result.current.design).toBe('classic')
    expect(document.documentElement.classList.contains(DESIGN_CLASS)).toBe(false)
    expect(localStorage.getItem(DESIGN_STORAGE_KEY)).toBe('classic')
  })

  it('updates every reader when another one switches', async () => {
    const a = renderHook(() => useDesign())
    const b = renderHook(() => useDesign())

    await act(async () => { a.result.current.setDesign('depot') })

    expect(b.result.current.design).toBe('depot')
  })
})

describe('initDesign', () => {
  it('applies the stored choice', () => {
    localStorage.setItem(DESIGN_STORAGE_KEY, 'classic')
    initDesign('depot')
    expect(document.documentElement.classList.contains(DESIGN_CLASS)).toBe(false)
  })

  it('applies the build default when nothing is stored', () => {
    initDesign('depot')
    expect(document.documentElement.classList.contains(DESIGN_CLASS)).toBe(true)
  })
})

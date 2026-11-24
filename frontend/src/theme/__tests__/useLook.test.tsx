// @vitest-environment jsdom
import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { LOOK_STORAGE_KEY } from '../lookPreference'
import { applyLook, initLook, setShopDefaultLook, useLook } from '../useLook'

const root = () => document.documentElement.classList

function reset() {
  localStorage.clear()
  setShopDefaultLook(null)
  root().remove('look-wallet', 'look-graphite')
}

beforeEach(reset)
afterEach(reset)

describe('useLook', () => {
  it('reads the look applied to the document', () => {
    applyLook('graphite')
    const { result } = renderHook(() => useLook())
    expect(result.current.look).toBe('graphite')
  })

  it('reads the default look when no look class is applied', () => {
    const { result } = renderHook(() => useLook())
    expect(result.current.look).toBe('wallet')
  })

  it('switches the document look, one class at a time, and remembers the choice', async () => {
    const { result } = renderHook(() => useLook())

    await act(async () => { result.current.setLook('graphite') })

    expect(result.current.look).toBe('graphite')
    expect(root().contains('look-graphite')).toBe(true)
    expect(root().contains('look-wallet')).toBe(false)
    expect(localStorage.getItem(LOOK_STORAGE_KEY)).toBe('graphite')

    await act(async () => { result.current.setLook('wallet') })

    expect(result.current.look).toBe('wallet')
    expect(root().contains('look-wallet')).toBe(true)
    expect(root().contains('look-graphite')).toBe(false)
  })

  it('updates every reader when another one switches', async () => {
    const a = renderHook(() => useLook())
    const b = renderHook(() => useLook())

    await act(async () => { a.result.current.setLook('graphite') })

    expect(b.result.current.look).toBe('graphite')
  })
})

describe('shop default look', () => {
  it('applies the shop default when this browser has no choice of its own', () => {
    initLook()
    setShopDefaultLook('graphite')
    expect(root().contains('look-graphite')).toBe(true)
  })

  it('leaves a personal choice alone', () => {
    localStorage.setItem(LOOK_STORAGE_KEY, 'wallet')
    initLook()
    setShopDefaultLook('graphite')
    expect(root().contains('look-wallet')).toBe(true)
  })

  it('goes back to the shop default when the person clears their choice', async () => {
    setShopDefaultLook('graphite')
    const { result } = renderHook(() => useLook())
    await act(async () => { result.current.setLook('wallet') })
    expect(result.current.personal).toBe('wallet')

    await act(async () => { result.current.setLook(null) })

    expect(result.current.personal).toBeNull()
    expect(result.current.look).toBe('graphite')
    expect(localStorage.getItem(LOOK_STORAGE_KEY)).toBeNull()
  })
})

describe('initLook', () => {
  it('applies the stored choice', () => {
    localStorage.setItem(LOOK_STORAGE_KEY, 'graphite')
    initLook()
    expect(root().contains('look-graphite')).toBe(true)
  })

  it('applies the default look when nothing is stored', () => {
    initLook()
    expect(root().contains('look-wallet')).toBe(true)
  })
})

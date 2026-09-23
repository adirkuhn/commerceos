// @vitest-environment jsdom
import { renderHook, act } from '@testing-library/react'
import type { ReactNode } from 'react'
import { MemoryRouter, useLocation } from 'react-router-dom'
import { describe, it, expect } from 'vitest'
import { useReportPeriod } from '../useReportPeriod'

const at = (url: string) => ({ children }: { children: ReactNode }) => <MemoryRouter initialEntries={[url]}>{children}</MemoryRouter>

describe('useReportPeriod', () => {
  it('defaults to 7D', () => {
    expect(renderHook(() => useReportPeriod(), { wrapper: at('/r') }).result.current[0]).toBe('7D')
  })

  it('reads ?period= so a refresh keeps the chosen period', () => {
    expect(renderHook(() => useReportPeriod(), { wrapper: at('/r?period=90D') }).result.current[0]).toBe('90D')
  })

  it('ignores an unknown period', () => {
    expect(renderHook(() => useReportPeriod(), { wrapper: at('/r?period=5Y') }).result.current[0]).toBe('7D')
  })

  it('writes ?period=, omitting the default', () => {
    const { result } = renderHook(() => ({ p: useReportPeriod(), loc: useLocation() }), { wrapper: at('/r') })
    act(() => result.current.p[1]('12M'))
    expect(result.current.loc.search).toBe('?period=12M')
    act(() => result.current.p[1]('7D'))
    expect(result.current.loc.search).toBe('')
  })
})

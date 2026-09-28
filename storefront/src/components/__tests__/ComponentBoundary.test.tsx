// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ComponentBoundary } from '../ComponentBoundary'

function Broken(): never {
  throw new Error('internal server error')
}

describe('ComponentBoundary', () => {
  it('contains a failing component and shows its fallback (nothing by default)', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => undefined)
    const { container } = render(<><ComponentBoundary nodeId="grid"><Broken /></ComponentBoundary><p>rest of page</p></>)
    expect(screen.getByText('rest of page')).toBeInTheDocument()
    expect(container.textContent).toBe('rest of page')

    render(<ComponentBoundary nodeId="grid" fallback={<span>could not load</span>}><Broken /></ComponentBoundary>)
    expect(screen.getByText('could not load')).toBeInTheDocument()
    expect(spy).toHaveBeenCalledWith('storefront: component grid failed', expect.any(Error))
    spy.mockRestore()
  })
})

// @vitest-environment jsdom
import { describe, it, expect } from 'vitest'
import { act } from '@testing-library/react'
import { ReactMounts } from '../reactMount'

describe('ReactMounts', () => {
  it('re-renders a component in place when its props change', async () => {
    const mounts = new ReactMounts<object>()
    const host = document.createElement('div')
    const key = {}
    await act(async () => mounts.render(key, host, <h2>Heading</h2>))
    await act(async () => mounts.render(key, host, <h2>Welcome</h2>))
    expect(host.querySelectorAll('h2')).toHaveLength(1)
    expect(host.textContent).toBe('Welcome')
  })

  // GrapesJS re-renders a view by emptying its element; the old React
  // tree must be let go without React trying to remove nodes that are gone.
  it('survives the host being emptied and mounts again', async () => {
    const mounts = new ReactMounts<object>()
    const host = document.createElement('div')
    const key = {}
    await act(async () => mounts.render(key, host, <p>one</p>))
    host.innerHTML = ''
    await act(async () => mounts.render(key, host, <p>two</p>))
    expect(host.textContent).toBe('two')
    await act(async () => mounts.unmount(key))
    expect(host.textContent).toBe('')
  })

  it('does not affect layout: the React container uses display: contents', async () => {
    const mounts = new ReactMounts<object>()
    const host = document.createElement('div')
    await act(async () => mounts.render({}, host, <span>x</span>))
    expect((host.firstElementChild as HTMLElement).style.display).toBe('contents')
  })
})

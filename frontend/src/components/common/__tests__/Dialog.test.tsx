// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { describe, it, expect, vi } from 'vitest'
import { Dialog } from '../Dialog'

function Harness({ onClose = vi.fn() }: { onClose?: () => void }) {
  const [open, setOpen] = useState(false)
  return (
    <>
      <button onClick={() => setOpen(true)}>Open</button>
      {open && (
        <Dialog onClose={() => { onClose(); setOpen(false) }}>
          <h3>Edit rule</h3>
          <input aria-label="Name" />
          <button>Save</button>
        </Dialog>
      )}
    </>
  )
}

describe('Dialog', () => {
  it('is a modal dialog named by its first heading', async () => {
    render(<Harness />)
    await userEvent.click(screen.getByRole('button', { name: 'Open' }))

    const dialog = screen.getByRole('dialog', { name: 'Edit rule' })
    expect(dialog).toHaveAttribute('aria-modal', 'true')
  })

  it('uses an explicit label over the heading', () => {
    render(<Dialog onClose={vi.fn()} label="Pair scanner"><h3>Scan</h3></Dialog>)
    expect(screen.getByRole('dialog', { name: 'Pair scanner' })).toBeInTheDocument()
  })

  it('moves focus into the dialog when it opens', async () => {
    render(<Harness />)
    await userEvent.click(screen.getByRole('button', { name: 'Open' }))
    expect(screen.getByRole('textbox', { name: 'Name' })).toHaveFocus()
  })

  it('leaves focus alone when content already focused itself (autoFocus)', () => {
    render(
      <Dialog onClose={vi.fn()}>
        <input aria-label="First" />
        <input aria-label="Search" autoFocus />
      </Dialog>,
    )
    expect(screen.getByRole('textbox', { name: 'Search' })).toHaveFocus()
  })

  it('closes on Escape and returns focus to the control that opened it', async () => {
    const onClose = vi.fn()
    render(<Harness onClose={onClose} />)
    const opener = screen.getByRole('button', { name: 'Open' })
    await userEvent.click(opener)

    await userEvent.keyboard('{Escape}')

    expect(onClose).toHaveBeenCalledTimes(1)
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(opener).toHaveFocus()
  })

  it('keeps Tab focus inside the dialog', async () => {
    render(<Harness />)
    await userEvent.click(screen.getByRole('button', { name: 'Open' }))
    const name = screen.getByRole('textbox', { name: 'Name' })
    const save = screen.getByRole('button', { name: 'Save' })

    await userEvent.tab()
    expect(save).toHaveFocus()
    await userEvent.tab()
    expect(name).toHaveFocus()
    await userEvent.tab({ shift: true })
    expect(save).toHaveFocus()
  })

  it('only the innermost of nested dialogs closes on Escape', async () => {
    const outerClose = vi.fn()
    const innerClose = vi.fn()
    render(
      <Dialog onClose={outerClose} label="Outer">
        <button>Outer action</button>
        <Dialog onClose={innerClose} label="Inner">
          <button>Inner action</button>
        </Dialog>
      </Dialog>,
    )
    screen.getByRole('button', { name: 'Inner action' }).focus()

    await userEvent.keyboard('{Escape}')

    expect(innerClose).toHaveBeenCalledTimes(1)
    expect(outerClose).not.toHaveBeenCalled()
  })

  it('does not close on Escape while dismissal is disabled (e.g. saving)', async () => {
    const onClose = vi.fn()
    render(<Dialog onClose={onClose} label="Saving" dismissible={false}><button>Wait</button></Dialog>)
    await userEvent.keyboard('{Escape}')
    expect(onClose).not.toHaveBeenCalled()
  })

  it('applies the overlay className', () => {
    render(<Dialog onClose={vi.fn()} label="X" className="fixed inset-0 z-50">x</Dialog>)
    expect(screen.getByRole('dialog')).toHaveClass('fixed', 'inset-0', 'z-50')
  })
})

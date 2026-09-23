// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi } from 'vitest'
import { useConfirm, type ConfirmOptions } from '../useConfirm'

const DELETE_RULE: ConfirmOptions = {
  title: 'Delete rule "Welcome"?',
  message: 'This cannot be undone.',
  confirmLabel: 'Delete rule',
  destructive: true,
}

function Harness({ onResult }: { onResult: (ok: boolean) => void }) {
  const { confirm, confirmDialog } = useConfirm()
  return (
    <>
      <button onClick={async () => onResult(await confirm(DELETE_RULE))}>Delete</button>
      {confirmDialog}
    </>
  )
}

describe('useConfirm', () => {
  it('asks in an alert dialog named by the question, with the consequence and specific actions', async () => {
    render(<Harness onResult={vi.fn()} />)
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    const dialog = screen.getByRole('alertdialog', { name: 'Delete rule "Welcome"?' })
    expect(dialog).toHaveTextContent('This cannot be undone.')
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Delete rule' })).toBeInTheDocument()
  })

  it('resolves true when the action is confirmed and closes', async () => {
    const onResult = vi.fn()
    render(<Harness onResult={onResult} />)
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    await userEvent.click(screen.getByRole('button', { name: 'Delete rule' }))

    await waitFor(() => expect(onResult).toHaveBeenCalledWith(true))
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
  })

  it('resolves false on Cancel', async () => {
    const onResult = vi.fn()
    render(<Harness onResult={onResult} />)
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))

    await waitFor(() => expect(onResult).toHaveBeenCalledWith(false))
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
  })

  it('resolves false on Escape', async () => {
    const onResult = vi.fn()
    render(<Harness onResult={onResult} />)
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    await userEvent.keyboard('{Escape}')

    await waitFor(() => expect(onResult).toHaveBeenCalledWith(false))
  })

  it('has no default button: Return alone does not confirm', async () => {
    const onResult = vi.fn()
    render(<Harness onResult={onResult} />)
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    await userEvent.keyboard('{Enter}')

    expect(onResult).not.toHaveBeenCalled()
    expect(screen.getByRole('alertdialog')).toBeInTheDocument()
  })

  it('resolves false if the owner unmounts while asking', async () => {
    const onResult = vi.fn()
    const { unmount } = render(<Harness onResult={onResult} />)
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    unmount()

    await waitFor(() => expect(onResult).toHaveBeenCalledWith(false))
  })
})

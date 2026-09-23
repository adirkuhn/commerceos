import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

// Answers the in-app confirmation prompt (components/common/useConfirm).
export async function acceptConfirm(): Promise<HTMLElement> {
  const dialog = await screen.findByRole('alertdialog')
  const buttons = within(dialog).getAllByRole('button')
  await userEvent.click(buttons[buttons.length - 1])
  return dialog
}

export async function dismissConfirm(): Promise<HTMLElement> {
  const dialog = await screen.findByRole('alertdialog')
  await userEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }))
  return dialog
}

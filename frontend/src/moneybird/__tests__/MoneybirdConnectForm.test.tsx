// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MoneybirdConnectForm } from '../MoneybirdConnectForm'
import { CodedApiError } from '../../api/codedApiError'

vi.mock('../api', () => ({ connectMoneybird: vi.fn() }))
import * as api from '../api'

beforeEach(() => { vi.clearAllMocks() })

describe('MoneybirdConnectForm', () => {
  it('connects with the token, then closes and reloads', async () => {
    vi.mocked(api.connectMoneybird).mockResolvedValue({ connection_id: 'conn-1' })
    const onConnected = vi.fn()
    const onDone = vi.fn()
    render(<MoneybirdConnectForm token="t" onConnected={onConnected} onDone={onDone} />)

    await userEvent.type(screen.getByLabelText('Personal API token'), '  mb-token ')
    await userEvent.click(screen.getByRole('button', { name: /connect/i }))

    expect(api.connectMoneybird).toHaveBeenCalledWith('t', 'mb-token')
    expect(onDone).toHaveBeenCalled()
    expect(onConnected).toHaveBeenCalled()
  })

  it('shows a rejected token', async () => {
    vi.mocked(api.connectMoneybird).mockRejectedValue(new CodedApiError('MONEYBIRD_TOKEN_INVALID', 'x', undefined))
    const onDone = vi.fn()
    render(<MoneybirdConnectForm token="t" onConnected={vi.fn()} onDone={onDone} />)

    await userEvent.type(screen.getByLabelText('Personal API token'), 'bad')
    await userEvent.click(screen.getByRole('button', { name: /connect/i }))

    expect(await screen.findByText("Moneybird didn't accept the API token.")).toBeInTheDocument()
    expect(onDone).not.toHaveBeenCalled()
  })
})

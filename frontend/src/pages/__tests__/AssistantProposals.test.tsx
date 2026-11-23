// @vitest-environment jsdom
import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import Assistant from '../Assistant'
import * as api from '../../assistant/api'
import { CodedApiError } from '../../api/codedApiError'
import { formatMoney } from '../../format/money'
import {
  aConversionProposal, aDetail, aMessage, aProposal, aProposedMessage,
} from '../../assistant/__tests__/fixtures'
import { renderAt } from '../../test/renderAt'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: { role: 'customer_admin' }, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))
vi.mock('../../assistant/api')

beforeEach(() => {
  vi.mocked(api.getAssistantRequest).mockReset()
  vi.mocked(api.confirmAssistantProposal).mockReset()
  vi.mocked(api.rejectAssistantProposal).mockReset()
})

const opening = aMessage({ seq: 1, content: 'Quote 10 farofa for Ana Bakery' })

function showQuoteProposal(over = {}) {
  vi.mocked(api.getAssistantRequest).mockResolvedValue(
    aDetail({ status: 'awaiting_confirmation' }, [opening, aProposedMessage(2, 'p-1')], [aProposal(over)]),
  )
  return renderAt(<Assistant />, '/assistant', '/assistant/r-1')
}

async function card(): Promise<HTMLElement> {
  return screen.findByTestId('assistant-proposal-p-1')
}

describe('quote proposal', () => {
  it('shows what will be created, with names -- never bare ids', async () => {
    showQuoteProposal()
    const c = await card()

    expect(within(c).getByText('Proposed quote')).toBeInTheDocument()
    expect(within(c).getByTestId('proposal-customer')).toHaveTextContent('Ana Bakery')
    expect(within(c).getByTestId('proposal-customer')).toHaveTextContent('ana@bakery.test')
    expect(within(c).getByText('Farofa')).toBeInTheDocument()
    expect(within(c).getByText('1 kg · FAR-1')).toBeInTheDocument()
    expect(within(c).getByText('10% discount')).toBeInTheDocument()
    expect(within(c).getByText('Prices include VAT')).toBeInTheDocument()
    expect(within(c).getByTestId('proposal-total')).toHaveTextContent(formatMoney(4500, 'EUR'))
    expect(within(c).queryByText('p-1')).not.toBeInTheDocument()
    expect(within(c).queryByText('s-1')).not.toBeInTheDocument()
    expect(screen.getByText('Waiting for your confirmation')).toBeInTheDocument()
    expect(screen.getByLabelText('Ask for changes')).toBeInTheDocument()
  })

  it('confirm creates it and shows a link to the quote', async () => {
    vi.mocked(api.confirmAssistantProposal).mockResolvedValue(aDetail({ status: 'awaiting_confirmation' }, [
      opening, aProposedMessage(2, 'p-1'),
      aMessage({ seq: 3, role: 'assistant', content: 'The user confirmed…', notice: 'proposal_confirmed', proposal_id: 'p-1' }),
    ], [aProposal({ status: 'confirmed', result_id: 'q-1', result_ref: 'Q-0012' })]))
    showQuoteProposal()

    await userEvent.click(within(await card()).getByRole('button', { name: 'Confirm' }))

    expect(api.confirmAssistantProposal).toHaveBeenCalledWith('test-token', 'r-1', 'p-1', {})
    const link = await within(await card()).findByRole('link', { name: 'Created quote Q-0012' })
    expect(link).toHaveAttribute('href', '/quotes/q-1')
    expect(within(await card()).queryByRole('button', { name: 'Confirm' })).not.toBeInTheDocument()
    expect(screen.getByText('You confirmed the proposal.')).toBeInTheDocument()
    expect(screen.queryByText('The user confirmed…')).not.toBeInTheDocument()
  })

  it('reject asks first, then records it', async () => {
    vi.mocked(api.rejectAssistantProposal).mockResolvedValue(
      aDetail({ status: 'answered' }, [opening, aProposedMessage(2, 'p-1')], [aProposal({ status: 'rejected' })]),
    )
    showQuoteProposal()

    await userEvent.click(within(await card()).getByRole('button', { name: 'Reject' }))
    await userEvent.click(within(screen.getByRole('alertdialog')).getByRole('button', { name: 'Reject proposal' }))

    expect(api.rejectAssistantProposal).toHaveBeenCalledWith('test-token', 'r-1', 'p-1')
    expect(await within(await card()).findByText('Rejected')).toBeInTheDocument()
  })

  it('a failed confirm shows why in the card and keeps the buttons', async () => {
    vi.mocked(api.confirmAssistantProposal).mockRejectedValue(new CodedApiError('ASSISTANT_PROPOSAL_NOT_PENDING', 'x'))
    showQuoteProposal({ last_error: 'insufficient stock' })
    const c = await card()

    expect(within(c).getByText('insufficient stock')).toBeInTheDocument()
    await userEvent.click(within(c).getByRole('button', { name: 'Confirm' }))

    expect(await within(c).findByText('This proposal was already decided. Refresh to see what happened.')).toBeInTheDocument()
    expect(within(c).getByRole('button', { name: 'Confirm' })).toBeEnabled()
  })

  it('buttons are disabled while the confirm runs', async () => {
    vi.mocked(api.confirmAssistantProposal).mockReturnValue(new Promise(() => {}))
    showQuoteProposal()
    const c = await card()

    await userEvent.click(within(c).getByRole('button', { name: 'Confirm' }))

    expect(within(c).getByRole('button', { name: 'Confirming…' })).toBeDisabled()
    expect(within(c).getByRole('button', { name: 'Reject' })).toBeDisabled()
  })

  it.each([
    ['superseded', 'Replaced by your reply'],
    ['executing', 'Running…'],
  ] as const)('%s proposals show their state without actions', async (status, label) => {
    showQuoteProposal({ status })
    const c = await card()

    expect(within(c).getByText(label)).toBeInTheDocument()
    expect(within(c).queryByRole('button', { name: 'Confirm' })).not.toBeInTheDocument()
  })

  it('no actions once the request is no longer waiting on it', async () => {
    vi.mocked(api.getAssistantRequest).mockResolvedValue(
      aDetail({ status: 'cancelled' }, [opening, aProposedMessage(2, 'p-1')], [aProposal()]),
    )
    renderAt(<Assistant />, '/assistant', '/assistant/r-1')

    expect(within(await card()).queryByRole('button', { name: 'Confirm' })).not.toBeInTheDocument()
  })
})

describe('conversion proposal', () => {
  function showConversion() {
    vi.mocked(api.getAssistantRequest).mockResolvedValue(aDetail({ status: 'awaiting_confirmation' }, [
      opening, aProposedMessage(2, 'p-2'),
    ], [aConversionProposal()]))
    return renderAt(<Assistant />, '/assistant', '/assistant/r-1')
  }

  it('shows the quote, the lines, the warehouse by name and the stock shortfall', async () => {
    showConversion()
    const c = await screen.findByTestId('assistant-proposal-p-2')

    expect(within(c).getByText('Convert Q-0012 to a sales order')).toBeInTheDocument()
    expect(within(c).getByLabelText('Warehouse')).toHaveDisplayValue('Main warehouse')
    const short = within(c).getByTestId('proposal-shortfall')
    expect(short).toHaveTextContent('Farofa')
    expect(short).toHaveTextContent('FAR-1')
    expect(short).toHaveTextContent('10 requested, 4 available, 6 short')
  })

  it('confirms with the chosen warehouse and links to the order', async () => {
    vi.mocked(api.confirmAssistantProposal).mockResolvedValue(aDetail({ status: 'answered' }, [
      opening, aProposedMessage(2, 'p-2'),
    ], [aConversionProposal({ status: 'confirmed', result_id: 'so-1', result_ref: 'CO-40' })]))
    showConversion()
    const c = await screen.findByTestId('assistant-proposal-p-2')

    await userEvent.selectOptions(within(c).getByLabelText('Warehouse'), 'Overflow')
    await userEvent.click(within(c).getByRole('button', { name: 'Confirm' }))

    expect(api.confirmAssistantProposal).toHaveBeenCalledWith('test-token', 'r-1', 'p-2', { warehouse_id: 'w-2' })
    const link = await within(await screen.findByTestId('assistant-proposal-p-2')).findByRole('link', { name: 'Created order CO-40' })
    expect(link).toHaveAttribute('href', '/sales-orders/so-1')
    await waitFor(() => expect(screen.getByText('Answered')).toBeInTheDocument())
  })
})

describe('decisions in the conversation', () => {
  it('rejected notice is a translated system line', async () => {
    vi.mocked(api.getAssistantRequest).mockResolvedValue(aDetail({ status: 'answered' }, [
      opening, aProposedMessage(2, 'p-1'),
      aMessage({ seq: 3, role: 'assistant', content: 'The user rejected…', notice: 'proposal_rejected', proposal_id: 'p-1' }),
    ], [aProposal({ status: 'rejected' })]))
    renderAt(<Assistant />, '/assistant', '/assistant/r-1')

    expect(await screen.findByText('You rejected the proposal.')).toBeInTheDocument()
    expect(screen.queryByText('The user rejected…')).not.toBeInTheDocument()
  })
})

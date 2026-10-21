// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { DefaultShippingProviderSetting } from '../DefaultShippingProviderSetting'
import { CodedApiError } from '../../api/codedApiError'
import { shippingConnectionsOf } from '../../test/shippingConnections'

vi.mock('../api', () => ({ listShippingConnections: vi.fn(), updateShippingPreferences: vi.fn() }))
import * as shippingApi from '../api'

const TWO = [{ id: 'mp', provider: 'myparcel' }, { id: 'pn', provider: 'postnl' }]

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(shippingApi.listShippingConnections).mockResolvedValue(shippingConnectionsOf(TWO))
})

async function renderSetting(canEdit = true) {
  render(<DefaultShippingProviderSetting token="t" canEdit={canEdit} />)
  return screen.findByLabelText('Default shipping provider')
}

describe('DefaultShippingProviderSetting', () => {
  it('lists Automatic plus each shipping connection, on Automatic when nothing was picked', async () => {
    const select = await renderSetting()
    expect(select).toHaveValue('')
    expect(screen.getAllByRole('option').map(o => o.textContent)).toEqual([
      'Automatic – oldest connection (MyParcel)', 'MyParcel', 'PostNL',
    ])
  })

  it('shows the explicit pick', async () => {
    vi.mocked(shippingApi.listShippingConnections).mockResolvedValue(shippingConnectionsOf(TWO, 'pn', 'pn'))
    expect(await renderSetting()).toHaveValue('pn')
  })

  it('an admin picking a provider saves only that field', async () => {
    vi.mocked(shippingApi.updateShippingPreferences).mockResolvedValue(shippingConnectionsOf(TWO, 'pn', 'pn'))
    const select = await renderSetting()
    await userEvent.selectOptions(select, 'pn')
    expect(shippingApi.updateShippingPreferences).toHaveBeenCalledWith('t', 'pn')
    expect(await screen.findByText('Default shipping provider saved.')).toBeInTheDocument()
    expect(select).toHaveValue('pn')
  })

  it('Automatic saves null', async () => {
    vi.mocked(shippingApi.listShippingConnections).mockResolvedValue(shippingConnectionsOf(TWO, 'pn', 'pn'))
    vi.mocked(shippingApi.updateShippingPreferences).mockResolvedValue(shippingConnectionsOf(TWO))
    const select = await renderSetting()
    await userEvent.selectOptions(select, '')
    expect(shippingApi.updateShippingPreferences).toHaveBeenCalledWith('t', null)
    await waitFor(() => expect(select).toHaveValue(''))
  })

  it('is read-only for a non-admin', async () => {
    const select = await renderSetting(false)
    expect(select).toBeDisabled()
    expect(screen.getByText('Only an owner or admin can change this.')).toBeInTheDocument()
  })

  it('shows why a save failed and keeps the old value', async () => {
    vi.mocked(shippingApi.updateShippingPreferences).mockRejectedValue(new CodedApiError('NOT_A_SHIPPING_CONNECTION', 'nope', undefined))
    const select = await renderSetting()
    await userEvent.selectOptions(select, 'pn')
    expect(await screen.findByRole('alert')).toHaveTextContent('This connection can’t create shipping labels.')
    expect(select).toHaveValue('')
    expect(screen.queryByText('Default shipping provider saved.')).not.toBeInTheDocument()
  })

  it('shows a load failure', async () => {
    vi.mocked(shippingApi.listShippingConnections).mockRejectedValue(new Error('down'))
    render(<DefaultShippingProviderSetting token="t" canEdit />)
    expect(await screen.findByText('down')).toBeInTheDocument()
  })

  it('is hidden without shipping connections', async () => {
    vi.mocked(shippingApi.listShippingConnections).mockResolvedValue(shippingConnectionsOf([]))
    const { container } = render(<DefaultShippingProviderSetting token="t" canEdit />)
    await waitFor(() => expect(shippingApi.listShippingConnections).toHaveBeenCalled())
    await waitFor(() => expect(container).toBeEmptyDOMElement())
  })
})

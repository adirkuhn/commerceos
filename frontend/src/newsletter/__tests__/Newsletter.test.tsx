// @vitest-environment jsdom
import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import i18next from 'i18next'
import Newsletter from '../../pages/Newsletter'
import { renderAt } from '../../test/renderAt'
import type { Subscriber } from '../types'

vi.mock('../../auth/AuthContext', () => ({ useAuth: () => ({ token: 'tok' }) }))
vi.mock('../api', () => ({ listSubscribers: vi.fn(), exportSubscribersCSV: vi.fn(), eraseSubscriber: vi.fn() }))
vi.mock('../download', () => ({ downloadBlob: vi.fn() }))
import * as api from '../api'
import { downloadBlob } from '../download'

const renderPage = (url = '/newsletter') => renderAt(<Newsletter />, '/newsletter', url)

const sub = (o: Partial<Subscriber>): Subscriber => ({
  id: 's1', email: 'ann@x.test', status: 'confirmed', consent_at: '2026-09-01T10:00:00Z',
  confirmed_at: '2026-09-01T10:05:00Z', created_at: '2026-09-01T10:00:00Z', ...o,
})

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.listSubscribers).mockResolvedValue({ items: [sub({}), sub({ id: 's2', email: 'bob@x.test', status: 'pending', confirmed_at: null })], total: 2 })
})

describe('Newsletter', () => {
  it('lists subscribers with their status', async () => {
    renderPage()
    const ann = (await screen.findByText('ann@x.test')).closest('tr')!
    expect(within(ann).getByText('Subscribed')).toBeInTheDocument()
    const bob = screen.getByText('bob@x.test').closest('tr')!
    expect(within(bob).getByText('Awaiting confirmation')).toBeInTheDocument()
    expect(api.listSubscribers).toHaveBeenCalledWith('tok', { status: undefined, search: undefined, page: 1, perPage: 50 })
  })

  it('filters by status and search from the URL', async () => {
    renderPage('/newsletter?status=unsubscribed&q=ann&page=2')
    await screen.findByText('ann@x.test')
    expect(api.listSubscribers).toHaveBeenLastCalledWith('tok', { status: 'unsubscribed', search: 'ann', page: 2, perPage: 50 })
    expect(screen.getByLabelText('Search subscribers')).toHaveValue('ann')
  })

  it('searches once typing pauses and starts from the first page', async () => {
    renderPage('/newsletter?page=3')
    await screen.findByText('ann@x.test')
    await userEvent.type(screen.getByLabelText('Search subscribers'), 'bob')
    await waitFor(() => expect(api.listSubscribers).toHaveBeenLastCalledWith('tok', { status: undefined, search: 'bob', page: 1, perPage: 50 }))
  })

  it('exports confirmed subscribers as a CSV download', async () => {
    const blob = new Blob(['email,confirmed_at\n'])
    vi.mocked(api.exportSubscribersCSV).mockResolvedValue(blob)
    renderPage()
    await userEvent.click(await screen.findByRole('button', { name: /Export subscribed/ }))
    await waitFor(() => expect(downloadBlob).toHaveBeenCalledWith(blob, 'newsletter-subscribers.csv'))
  })

  it('shows export failures', async () => {
    vi.mocked(api.exportSubscribersCSV).mockRejectedValue(new Error('forbidden'))
    renderPage()
    await userEvent.click(await screen.findByRole('button', { name: /Export subscribed/ }))
    expect(await screen.findByRole('alert')).toHaveTextContent('forbidden')
  })

  it('erases a subscriber after confirmation', async () => {
    vi.mocked(api.eraseSubscriber).mockResolvedValue()
    renderPage()
    await userEvent.click(await screen.findByRole('button', { name: 'Erase ann@x.test' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Erase subscriber' }))
    await waitFor(() => expect(api.eraseSubscriber).toHaveBeenCalledWith('tok', 's1'))
    expect(api.listSubscribers).toHaveBeenCalledTimes(2)
  })

  it('shows load errors', async () => {
    vi.mocked(api.listSubscribers).mockRejectedValue(new Error('boom'))
    renderPage()
    expect(await screen.findByRole('alert')).toHaveTextContent('boom')
  })
})

describe('in Brazilian Portuguese', () => {
  afterEach(async () => { await i18next.changeLanguage('en') })

  it('lists subscribers with their status in pt-BR', async () => {
    await i18next.changeLanguage('pt-BR')
    renderPage()
    const ann = (await screen.findByText('ann@x.test')).closest('tr')!
    expect(within(ann).getByText('Inscrito')).toBeInTheDocument()
  })
})

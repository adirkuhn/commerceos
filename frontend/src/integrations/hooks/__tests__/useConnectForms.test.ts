// @vitest-environment jsdom
import { renderHook, act } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import type React from 'react'

vi.mock('../../api', () => ({
  startWooConnect: vi.fn(),
  manualConnectWooCommerce: vi.fn(),
  connectWhatsapp: vi.fn(),
  connectStripe: vi.fn(),
  connectRevolut: vi.fn(),
  connectEmail: vi.fn(),
  setWebhookSecret: vi.fn(),
}))

import * as api from '../../api'
import { useConnectForms } from '../useConnectForms'
import type { Connection } from '../../types'

const event = () => ({ preventDefault: vi.fn() }) as unknown as React.FormEvent
const conn = (id: string): Connection => ({ id, provider: 'stripe', store_url: '', status: 'connected', created_at: '' })

function setup(token = 'tok') {
  const onConnected = vi.fn()
  const onClose = vi.fn()
  const hook = renderHook(() => useConnectForms(token, onConnected, onClose))
  return { hook, onConnected, onClose }
}

beforeEach(() => vi.clearAllMocks())

describe('useConnectForms', () => {
  it('starts with empty fields, auto Woo mode and port 587', () => {
    const { hook } = setup()
    expect(hook.result.current.woo.mode).toBe('auto')
    expect(hook.result.current.email.port).toBe('587')
    expect(hook.result.current.error).toBe('')
    expect(hook.result.current.paymentWebhook.webhook).toBeNull()
  })

  it('does nothing without a token', async () => {
    const { hook } = setup('')
    await act(() => hook.result.current.stripe.connect(event()))
    expect(api.connectStripe).not.toHaveBeenCalled()
  })

  it('WhatsApp connect stores the one-time webhook URL and reloads', async () => {
    vi.mocked(api.connectWhatsapp).mockResolvedValue({ connection: conn('wa'), webhook_url: 'https://hook' })
    const { hook, onConnected } = setup()
    act(() => {
      hook.result.current.whatsapp.setBaseUrl('https://evo')
      hook.result.current.whatsapp.setInstanceName('inst')
      hook.result.current.whatsapp.setApiKey('key')
    })
    await act(() => hook.result.current.whatsapp.connect(event()))

    expect(api.connectWhatsapp).toHaveBeenCalledWith('tok', { baseUrl: 'https://evo', instanceName: 'inst', apiKey: 'key' })
    expect(hook.result.current.whatsapp.webhookUrl).toBe('https://hook')
    expect(hook.result.current.starting).toBe(false)
    expect(onConnected).toHaveBeenCalledOnce()
  })

  it('Stripe connect switches to the webhook step, then saves the pasted secret', async () => {
    vi.mocked(api.connectStripe).mockResolvedValue({ connection: conn('s1'), webhook_url: 'https://stripe-hook' })
    vi.mocked(api.setWebhookSecret).mockResolvedValue({ connection_id: 's1', webhook_secret: 'whsec' })
    const { hook } = setup()
    act(() => hook.result.current.stripe.setSecretKey('sk'))
    await act(() => hook.result.current.stripe.connect(event()))
    expect(hook.result.current.paymentWebhook.webhook).toEqual({ connectionId: 's1', url: 'https://stripe-hook' })

    act(() => hook.result.current.paymentWebhook.setSecret('whsec'))
    await act(() => hook.result.current.paymentWebhook.save(event()))
    expect(api.setWebhookSecret).toHaveBeenCalledWith('tok', 's1', 'whsec')
    expect(hook.result.current.paymentWebhook.saved).toBe(true)
  })

  it('surfaces a webhook-secret save failure', async () => {
    vi.mocked(api.connectRevolut).mockResolvedValue({ connection: conn('r1'), webhook_url: 'u' })
    vi.mocked(api.setWebhookSecret).mockRejectedValue(new Error('nope'))
    const { hook } = setup()
    await act(() => hook.result.current.revolut.connect(event()))
    await act(() => hook.result.current.paymentWebhook.save(event()))
    expect(hook.result.current.paymentWebhook.error).toBe('nope')
    expect(hook.result.current.paymentWebhook.saving).toBe(false)
  })

  it('Email connect closes the catalog and reloads', async () => {
    vi.mocked(api.connectEmail).mockResolvedValue(conn('e1'))
    const { hook, onClose, onConnected } = setup()
    await act(() => hook.result.current.email.connect(event()))
    expect(onClose).toHaveBeenCalledOnce()
    expect(onConnected).toHaveBeenCalledOnce()
  })

  it('a failed connect sets the shared error and clears starting', async () => {
    vi.mocked(api.connectRevolut).mockRejectedValue(new Error('invalid key'))
    const { hook, onConnected } = setup()
    await act(() => hook.result.current.revolut.connect(event()))
    expect(hook.result.current.error).toBe('invalid key')
    expect(hook.result.current.starting).toBe(false)
    expect(onConnected).not.toHaveBeenCalled()
  })

  it('uses a fallback message for non-Error rejections', async () => {
    vi.mocked(api.startWooConnect).mockRejectedValue('boom')
    const { hook } = setup()
    await act(() => hook.result.current.woo.startConnect(event()))
    expect(hook.result.current.error).toBe('Failed to start connection')
  })

  it('manual Woo connect stays busy after success (the catalog is closing)', async () => {
    vi.mocked(api.manualConnectWooCommerce).mockResolvedValue(conn('w1'))
    const { hook, onClose, onConnected } = setup()
    act(() => {
      hook.result.current.woo.setMode('manual')
      hook.result.current.woo.setStoreUrl('https://s')
      hook.result.current.woo.setConsumerKey('ck')
      hook.result.current.woo.setConsumerSecret('cs')
    })
    await act(() => hook.result.current.woo.manualConnect(event()))
    expect(api.manualConnectWooCommerce).toHaveBeenCalledWith('tok', { storeUrl: 'https://s', consumerKey: 'ck', consumerSecret: 'cs' })
    expect(onConnected).toHaveBeenCalledOnce()
    expect(onClose).toHaveBeenCalledOnce()
    expect(hook.result.current.starting).toBe(true)
  })
})

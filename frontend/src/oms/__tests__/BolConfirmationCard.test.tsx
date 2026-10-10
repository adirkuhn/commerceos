// @vitest-environment jsdom
import i18next from 'i18next'
import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { BolConfirmationCard } from '../BolConfirmationCard'
import type { BolShipmentConfirmation } from '../bolConfirmations'

function confirmation(overrides: Partial<BolShipmentConfirmation> = {}): BolShipmentConfirmation {
  return {
    id: 'conf-1', shipment_id: 'ship-1', external_order_id: 'C000HLRF87', action: 'create_shipment', transport_id: '', transporter_code: 'TNT',
    track_and_trace: '3SMYPA859699114', process_status_id: 'ps-1', status: 'pending', error_message: '',
    needs_manual_action: false, created_at: '2026-10-07T14:11:00Z', resolved_at: null,
    ...overrides,
  }
}

describe('BolConfirmationCard', () => {
  it('says the shipment was sent and Bol is still processing it', () => {
    render(<BolConfirmationCard confirmations={[confirmation()]} error={null} />)

    expect(screen.getByText('Sent to Bol, waiting for Bol to confirm')).toBeInTheDocument()
    expect(screen.getByText(/TNT · 3SMYPA859699114/)).toBeInTheDocument()
  })

  it('says Bol confirmed the shipment', () => {
    render(<BolConfirmationCard confirmations={[confirmation({ status: 'success', resolved_at: '2026-10-07T14:13:00Z' })]} error={null} />)

    expect(screen.getByText('Confirmed on Bol')).toHaveClass('text-success')
  })

  it('makes a rejected shipment unmistakable: Bol\'s reason and what to do', () => {
    render(<BolConfirmationCard confirmations={[confirmation({
      status: 'failure', needs_manual_action: true, error_message: 'Invalid track and trace code.',
    })]} error={null} />)

    expect(screen.getByRole('alert')).toBeInTheDocument()
    expect(screen.getByText('Bol did not accept the shipment')).toHaveClass('text-danger')
    expect(screen.getByText('Invalid track and trace code.')).toBeInTheDocument()
    expect(screen.getByText('The order is still open on Bol. Confirm it in the Bol seller portal (order C000HLRF87).')).toBeInTheDocument()
  })

  it('treats a Bol timeout as needing manual confirmation too', () => {
    render(<BolConfirmationCard confirmations={[confirmation({ status: 'timeout', needs_manual_action: true })]} error={null} />)

    expect(screen.getByText('Bol did not answer in time')).toHaveClass('text-danger')
    expect(screen.getByRole('alert')).toBeInTheDocument()
  })

  it('says nothing was sent yet when there is no attempt', () => {
    render(<BolConfirmationCard confirmations={[]} error={null} />)

    expect(screen.getByText('Not sent to Bol yet')).toBeInTheDocument()
  })

  it('lists earlier attempts below the newest one', () => {
    render(<BolConfirmationCard confirmations={[
      confirmation({ id: 'new', status: 'success' }),
      confirmation({ id: 'old', status: 'failure', needs_manual_action: true, error_message: 'Bol was down.' }),
    ]} error={null} />)

    expect(screen.getByText('Confirmed on Bol')).toBeInTheDocument()
    expect(screen.getByText('Earlier attempts')).toBeInTheDocument()
    expect(screen.getByText(/Bol was down\./)).toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('describes a track & trace correction after a replaced label', () => {
    render(<BolConfirmationCard confirmations={[confirmation({ action: 'update_transport', transport_id: 't-1', status: 'success' })]} error={null} />)

    expect(screen.getByText('Track & trace corrected on Bol')).toHaveClass('text-success')
  })

  it('says Bol still has the old track & trace when a correction failed', () => {
    render(<BolConfirmationCard confirmations={[confirmation({
      action: 'update_transport', transport_id: 't-1', status: 'failure', needs_manual_action: true, error_message: 'Transport already delivered.',
    })]} error={null} />)

    expect(screen.getByRole('alert')).toBeInTheDocument()
    expect(screen.getByText('Bol did not accept the track & trace correction')).toHaveClass('text-danger')
    expect(screen.getByText('Bol still has the old track & trace. Update it in the Bol seller portal (order C000HLRF87).')).toBeInTheDocument()
  })

  it('shows a loading error', () => {
    render(<BolConfirmationCard confirmations={[]} error="sales order not found" />)

    expect(screen.getByText('sales order not found')).toHaveClass('text-danger')
  })

  it('explains a failure CommerceOS knows the reason for in the reader’s language', async () => {
    await i18next.changeLanguage('nl')
    try {
      render(<BolConfirmationCard error={null} confirmations={[confirmation({
        status: 'failure', needs_manual_action: true, failure_reason: 'no_transporter_code',
        error_message: 'Bol has no transporter code for Carrier 99 -- confirm this order in Bol yourself.',
      })]} />)
      expect(screen.getByText('Bol heeft geen vervoerderscode voor deze vervoerder, dus er is niets naar Bol gestuurd.')).toBeInTheDocument()
      expect(screen.queryByText(/Carrier 99/)).not.toBeInTheDocument()
    } finally {
      await i18next.changeLanguage('en')
    }
  })
})

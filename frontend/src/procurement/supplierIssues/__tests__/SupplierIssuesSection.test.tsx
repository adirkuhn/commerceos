// @vitest-environment jsdom
import { render, screen, within, waitFor, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('../../../auth/AuthContext', () => ({ useAuth: () => ({ token: 'tok' }) }))
vi.mock('../../api', () => ({ listSupplierIssues: vi.fn(), updateSupplierIssueStatus: vi.fn(), emailSupplierIssues: vi.fn() }))
vi.mock('../../../newsletter/download', () => ({ downloadBlob: vi.fn() }))

import * as api from '../../api'
import { downloadBlob } from '../../../newsletter/download'
import { SupplierIssuesSection } from '../SupplierIssuesSection'
import type { SupplierIssue, SupplierIssueList } from '../../types'
import { CodedApiError } from '../../../api/codedApiError'

const PO_ID = '3f6d1c2a-0b7e-4c1d-9a2f-5e8b7c6d4a31'

const missing: SupplierIssue = {
  source: 'booking_shortfall', source_id: 'line-1', kind: 'missing', product_name: 'Pão de queijo', sku_code: 'PDQ-1KG',
  supplier_sku: 'B-100', supplier_description: 'Pao de queijo 1kg', expected_qty: 10, received_qty: 7, quantity: 3,
  reason: '', reported_by: '', occurred_at: '2026-10-01T12:00:00Z', status: 'open', status_note: '',
}
const damaged: SupplierIssue = {
  source: 'booking_scan', source_id: 'scan-3', kind: 'damaged', product_name: 'Guaraná 2L', supplier_sku: '', supplier_description: 'Guarana 2L',
  quantity: 2, reason: 'crushed', reported_by: 'ana', occurred_at: '2026-10-01T09:00:00Z', status: 'claimed', status_note: 'emailed',
}
const resolved: SupplierIssue = {
  source: 'booking_scan', source_id: 'scan-4', kind: 'lost', supplier_sku: '', supplier_description: 'Farofa 500g',
  quantity: 1, reason: '', reported_by: '', occurred_at: '2026-10-01T09:00:00Z', status: 'resolved', status_note: 'CN-7',
}

function listOf(issues: SupplierIssue[], extra: Partial<SupplierIssueList> = {}): SupplierIssueList {
  return {
    purchase_order_id: PO_ID, booking_in_progress: false, issues,
    supplier: { id: 'sup-1', name: 'Bauducco NL', email: 'orders@bauducco.example' }, ...extra,
  }
}

function renderSection() {
  return render(<SupplierIssuesSection purchaseOrderId={PO_ID} orderDate="2026-09-20T00:00:00Z" hasCompletedBooking />)
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.listSupplierIssues).mockResolvedValue(listOf([missing, damaged, resolved]))
})

describe('SupplierIssuesSection', () => {
  it('lists unresolved issues with kind, item and quantity; resolved ones behind a toggle', async () => {
    renderSection()

    const row = await screen.findByRole('row', { name: /Pao de queijo 1kg/ })
    expect(within(row).getByText('Missing')).toBeTruthy()
    expect(within(row).getByText('3')).toBeTruthy()
    expect(within(row).getByText(/ordered 10, received 7/)).toBeTruthy()
    expect(screen.getByRole('row', { name: /Guarana 2L/ })).toBeTruthy()
    expect(screen.queryByRole('row', { name: /Farofa/ })).toBeNull()

    await userEvent.click(screen.getByRole('checkbox', { name: 'Show resolved (1)' }))

    expect(screen.getByRole('row', { name: /Farofa/ })).toBeTruthy()
  })

  // The hidden table header is position:absolute; without a positioned
  // section it escapes the page's scroll area and stretches the whole page
  // with empty space.
  it('keeps its visually hidden table header inside the section', async () => {
    const { container } = renderSection()
    await screen.findByRole('row', { name: /Pao de queijo 1kg/ })

    expect((container.firstElementChild as HTMLElement).className).toMatch(/\brelative\b/)
  })

  it('says when there is nothing to claim', async () => {
    vi.mocked(api.listSupplierIssues).mockResolvedValue(listOf([]))
    renderSection()

    expect(await screen.findByText(/Nothing to claim/)).toBeTruthy()
  })

  it('says issues come after the first booking', async () => {
    vi.mocked(api.listSupplierIssues).mockResolvedValue(listOf([]))
    render(<SupplierIssuesSection purchaseOrderId={PO_ID} orderDate="2026-09-20T00:00:00Z" hasCompletedBooking={false} />)

    expect(await screen.findByText(/once a delivery of this order has been booked/)).toBeTruthy()
  })

  it('warns while a delivery is still being booked', async () => {
    vi.mocked(api.listSupplierIssues).mockResolvedValue(listOf([missing], { booking_in_progress: true }))
    renderSection()

    expect(await screen.findByText(/being booked right now/)).toBeTruthy()
  })

  it('changes an issue’s status', async () => {
    vi.mocked(api.updateSupplierIssueStatus).mockResolvedValue({ ...missing, status: 'claimed' })
    renderSection()

    const select = await screen.findByRole('combobox', { name: 'Status of Pao de queijo 1kg' })
    await userEvent.selectOptions(select, 'claimed')

    expect(api.updateSupplierIssueStatus).toHaveBeenCalledWith('tok', PO_ID, missing, 'claimed', '')
    await waitFor(() => expect((select as HTMLSelectElement).value).toBe('claimed'))
  })

  it('shows a failed status change', async () => {
    vi.mocked(api.updateSupplierIssueStatus).mockRejectedValue(new CodedApiError('SUPPLIER_ISSUE_NOT_FOUND', 'gone'))
    renderSection()

    await userEvent.selectOptions(await screen.findByRole('combobox', { name: 'Status of Pao de queijo 1kg' }), 'resolved')

    expect(await screen.findByText('This item is not on the purchase order’s supplier issue list.')).toBeTruthy()
  })

  it('saves a note with the current status', async () => {
    vi.mocked(api.updateSupplierIssueStatus).mockResolvedValue({ ...damaged, status_note: 'CN-9' })
    renderSection()

    const row = await screen.findByRole('row', { name: /Guarana 2L/ })
    await userEvent.click(within(row).getByRole('button', { name: 'Edit note' }))
    const input = within(row).getByRole('textbox', { name: 'Note for Guarana 2L' })
    await userEvent.clear(input)
    await userEvent.type(input, 'CN-9{Enter}')

    expect(api.updateSupplierIssueStatus).toHaveBeenCalledWith('tok', PO_ID, damaged, 'claimed', 'CN-9')
  })

  it('copies the message for the selected issues only', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
    renderSection()

    await userEvent.click(await screen.findByRole('checkbox', { name: 'Select Guarana 2L' }))
    await userEvent.click(screen.getByRole('button', { name: 'Copy for supplier' }))

    expect(writeText).toHaveBeenCalledOnce()
    const text = writeText.mock.calls[0][0] as string
    expect(text).toContain('Hello Bauducco NL,')
    expect(text).toContain('- 3 × Pao de queijo 1kg (your code B-100)')
    expect(text).not.toContain('Guarana')
    expect(await screen.findByText('Copied')).toBeTruthy()
  })

  it('downloads the selected issues as CSV', async () => {
    renderSection()

    await userEvent.click(await screen.findByRole('button', { name: 'Download CSV' }))

    expect(downloadBlob).toHaveBeenCalledOnce()
    const [blob, fileName] = vi.mocked(downloadBlob).mock.calls[0]
    expect(fileName).toBe('supplier-issues-3F6D1C2A.csv')
    const csv = await new Promise<string>(resolve => {
      const reader = new FileReader()
      reader.onload = () => resolve(reader.result as string)
      reader.readAsText(blob as Blob)
    })
    expect(csv.split('\n')).toHaveLength(3)
  })

  it('emails the supplier with a prefilled, editable message and shows the result', async () => {
    vi.mocked(api.emailSupplierIssues).mockResolvedValue({ sent_to: 'orders@bauducco.example', claimed: 1 })
    renderSection()

    await userEvent.click(await screen.findByRole('button', { name: 'Email supplier' }))
    const dialog = screen.getByRole('dialog', { name: 'Email supplier' })
    expect((within(dialog).getByLabelText('To') as HTMLInputElement).value).toBe('orders@bauducco.example')
    expect((within(dialog).getByLabelText('Subject') as HTMLInputElement).value).toBe('Problems with purchase order #3F6D1C2A')
    const body = within(dialog).getByLabelText('Message') as HTMLTextAreaElement
    expect(body.value).toContain('- 2 × Guarana 2L — crushed')
    fireEvent.change(body, { target: { value: 'Edited message' } })
    await userEvent.click(within(dialog).getByRole('button', { name: 'Send' }))

    expect(api.emailSupplierIssues).toHaveBeenCalledWith('tok', PO_ID, {
      to: 'orders@bauducco.example', subject: 'Problems with purchase order #3F6D1C2A', body: 'Edited message',
      issues: [{ source: 'booking_shortfall', source_id: 'line-1' }, { source: 'booking_scan', source_id: 'scan-3' }],
    })
    expect(await screen.findByText('Sent to orders@bauducco.example. 1 item marked as claimed.')).toBeTruthy()
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('keeps the email dialog open with the server’s reason when sending fails', async () => {
    vi.mocked(api.emailSupplierIssues).mockRejectedValue(new CodedApiError('SUPPLIER_EMAIL_NOT_SENT', 'x', '550 mailbox unavailable'))
    renderSection()

    await userEvent.click(await screen.findByRole('button', { name: 'Email supplier' }))
    const dialog = screen.getByRole('dialog', { name: 'Email supplier' })
    await userEvent.click(within(dialog).getByRole('button', { name: 'Send' }))

    expect(await within(dialog).findByText(/The email to the supplier could not be sent/)).toBeTruthy()
    expect(screen.getByRole('dialog')).toBeTruthy()
  })

  it('asks for an address when the supplier has none', async () => {
    vi.mocked(api.listSupplierIssues).mockResolvedValue(listOf([missing], { supplier: { id: 's', name: 'X', email: '' } }))
    renderSection()

    await userEvent.click(await screen.findByRole('button', { name: 'Email supplier' }))

    expect(screen.getByText(/has no email address yet/)).toBeTruthy()
    expect((screen.getByRole('button', { name: 'Send' }) as HTMLButtonElement).disabled).toBe(true)
  })

  it('disables the actions with nothing selected', async () => {
    renderSection()

    await userEvent.click(await screen.findByRole('checkbox', { name: 'Select all' }))

    for (const name of ['Copy for supplier', 'Download CSV', 'Print / PDF', 'Email supplier']) {
      expect((screen.getByRole('button', { name }) as HTMLButtonElement).disabled).toBe(true)
    }
  })

  it('opens a printable view of the selected issues', async () => {
    const print = vi.spyOn(window, 'print').mockImplementation(() => {})
    renderSection()

    await userEvent.click(await screen.findByRole('button', { name: 'Print / PDF' }))

    expect(screen.getByRole('heading', { name: 'Supplier issues — purchase order #3F6D1C2A' })).toBeTruthy()
    expect(screen.getByText('Supplier: Bauducco NL')).toBeTruthy()
    expect(print).toHaveBeenCalled()
  })
})

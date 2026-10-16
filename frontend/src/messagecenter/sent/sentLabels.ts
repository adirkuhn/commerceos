import type { TFunction } from 'i18next'
import type { SentEmailSource, SentEmailStatus, SentEmailSummary } from '../types'

export const SENT_EMAIL_STATUSES: SentEmailStatus[] = ['sent', 'failed', 'skipped']
export const SENT_EMAIL_SOURCES: SentEmailSource[] = ['automation', 'notification', 'sign_in_link', 'newsletter', 'supplier_issues']

export const sentStatusLabels = (t: TFunction<'messages'>): Record<SentEmailStatus, string> => ({
  sent: t('sent.status.sent'), failed: t('sent.status.failed'), skipped: t('sent.status.skipped'),
})

export const sentSourceLabels = (t: TFunction<'messages'>): Record<SentEmailSource, string> => ({
  automation: t('sent.source.automation'), notification: t('sent.source.notification'),
  sign_in_link: t('sent.source.sign_in_link'), newsletter: t('sent.source.newsletter'),
  supplier_issues: t('sent.source.supplier_issues'),
})

// sentReason explains a failed or skipped email: skip codes are translated,
// a mail server's error is shown as it came.
export function sentReason(t: TFunction<'messages'>, e: Pick<SentEmailSummary, 'status' | 'reason'>): string {
  if (e.status === 'skipped') {
    if (e.reason === 'no_recipient') return t('sent.reason.noRecipient')
    if (e.reason === 'template_disabled') return t('sent.reason.templateDisabled')
  }
  return e.reason
}

export const STATUS_CLASSES: Record<SentEmailStatus, string> = {
  sent: 'text-success', failed: 'text-danger', skipped: 'text-ink-4',
}

// dayStart / dayEnd turn a YYYY-MM-DD filter into the API's [from, to)
// moments, in the viewer's time zone.
export function dayStart(day: string): string {
  return new Date(`${day}T00:00:00`).toISOString()
}

export function dayEnd(day: string): string {
  const d = new Date(`${day}T00:00:00`)
  d.setDate(d.getDate() + 1)
  return d.toISOString()
}

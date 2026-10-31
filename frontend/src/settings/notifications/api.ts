import { CodedApiError, codedRequest } from '../codedRequest'
import type { NotificationPreferences, NotificationPreferencesInput } from './types'
import { appEnv } from '../../config/appEnv'

function url(): string {
  return `${appEnv().VITE_API_BASE_URL}/settings/notification-preferences`
}

const RECIPIENT_CODES = new Set(['NOTIFICATION_RECIPIENT_INVALID', 'NOTIFICATION_TOO_MANY_RECIPIENTS', 'NOTIFICATION_RECIPIENTS_REQUIRED'])

export class NotificationPreferencesError extends CodedApiError {
  // The error is about the recipients list (shown on that field).
  readonly isRecipientError: boolean

  constructor(code: string, message: string) {
    super(code, message)
    this.name = 'NotificationPreferencesError'
    this.isRecipientError = RECIPIENT_CODES.has(code)
  }
}

async function send<T>(method: 'GET' | 'PUT', token: string, body?: unknown): Promise<T> {
  try {
    return await codedRequest<T>(method, url(), token, body)
  } catch (e) {
    if (e instanceof CodedApiError) throw new NotificationPreferencesError(e.code, e.message)
    throw e
  }
}

export function getNotificationPreferences(token: string): Promise<NotificationPreferences> {
  return send<NotificationPreferences>('GET', token)
}

export function updateNotificationPreferences(token: string, input: NotificationPreferencesInput): Promise<NotificationPreferences> {
  return send<NotificationPreferences>('PUT', token, input)
}

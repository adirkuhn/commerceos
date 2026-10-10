import i18next from 'i18next'
import type { Notification } from './types'

interface TypeText {
  title: string
  message: string
  messageOne?: string
  parts?: Record<string, string>
}

const TOKEN = /\{\{\s*(\w+)\s*\}\}/g

function typeText(type: string): TypeText | undefined {
  const lng = i18next.resolvedLanguage ?? i18next.language
  const key = `types.${type}`
  const found = i18next.getResource(lng, 'notifications', key) ?? i18next.getResource('en', 'notifications', key)
  return found && typeof found === 'object' && typeof (found as TypeText).title === 'string' ? (found as TypeText) : undefined
}

function fill(text: string, params: Record<string, unknown>): string {
  return text.replace(TOKEN, (_, name: string) => String(params[name]))
}

function tokens(text: string): string[] {
  return [...text.matchAll(TOKEN)].map(m => m[1])
}

// notificationText is a notification's title and message in the admin's
// current language: its type's translation (notifications.json,
// types.<type>) filled with its params -- the count == 1 form when count is
// 1, plus each optional part whose param is set. Same rules as the
// backend's notifications.Type.Render, which writes the emails. Falls back
// to the stored (English) text for unknown types, and for notifications
// from before translations that lack the params their text needs.
export function notificationText(n: Notification): { title: string; message: string } {
  const params = n.params ?? {}
  const text = typeText(n.type)
  if (!text) return { title: n.title, message: n.message }

  let message = text.messageOne && params.count === 1 ? text.messageOne : text.message
  for (const [param, part] of Object.entries(text.parts ?? {})) {
    const v = params[param]
    if (v !== undefined && v !== null && v !== '') message += part
  }
  const needed = tokens(text.title + message)
  if (!needed.every(name => name in params)) return { title: n.title, message: n.message }
  return { title: fill(text.title, params), message: fill(message, params) }
}

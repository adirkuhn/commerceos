// URLs for the Message Center page -- each tab, and the record open inside
// it, is addressable so a refresh or Back keeps the user in place:
//   /message-center                     Templates tab (default)
//   /message-center/rules|conversations other tabs
//   /message-center/templates/<new|id>  template editor
//   /message-center/rules/<new|id>      automation rule editor
//   /message-center/conversations/<e164>  selected WhatsApp thread
export const MESSAGE_CENTER_PATH = '/message-center'

export const MESSAGE_CENTER_TABS = ['templates', 'rules', 'conversations'] as const
export type MessageCenterTab = typeof MESSAGE_CENTER_TABS[number]
export const DEFAULT_MESSAGE_CENTER_TAB: MessageCenterTab = 'templates'

export function messageCenterTabPath(tab: MessageCenterTab): string {
  return tab === DEFAULT_MESSAGE_CENTER_TAB ? MESSAGE_CENTER_PATH : `${MESSAGE_CENTER_PATH}/${tab}`
}

export const TEMPLATES_PATH = `${MESSAGE_CENTER_PATH}/templates`
export const RULES_PATH = `${MESSAGE_CENTER_PATH}/rules`
export const CONVERSATIONS_PATH = `${MESSAGE_CENTER_PATH}/conversations`

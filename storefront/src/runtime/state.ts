import type { DehydratedState } from '@tanstack/react-query'

// What the server hands to the browser for hydration (embedded as JSON).
export interface PageState {
  dehydrated: DehydratedState
  config: { publicApiBaseURL: string; storeKey: string; previewToken?: string }
}

export const STATE_ELEMENT_ID = 'cos-state'

/** JSON safe to embed inside a <script> element. */
export function serializeState(state: PageState): string {
  return JSON.stringify(state).replace(/</g, '\\u003c').replace(/\u2028/g, '\\u2028').replace(/\u2029/g, '\\u2029')
}

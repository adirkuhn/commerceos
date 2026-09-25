// Wording for the two scanner fulfilment sessions, so the shared session
// pieces (lock banner, start-over dialog, missing-line cards) read right on
// either screen.
export type ScanActivity = 'picking' | 'packing'

export const SCAN_ACTIVITY_WORDS: Record<ScanActivity, { doing: string; done: string }> = {
  picking: { doing: 'picking', done: 'picked' },
  packing: { doing: 'packing', done: 'packed' },
}

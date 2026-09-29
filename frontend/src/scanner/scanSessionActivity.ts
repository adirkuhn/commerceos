// The two scanner fulfilment sessions. Shared session pieces (lock banner,
// start-over dialog, missing-line cards) pick a whole sentence per activity
// -- word-by-word composition doesn't translate.
export type ScanActivity = 'picking' | 'packing'

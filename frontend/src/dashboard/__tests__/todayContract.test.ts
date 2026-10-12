import { describe, expect, it } from 'vitest'
import dashboardSpec from '../../../../backend/api/dashboard/openapi.yaml?raw'
import type { TodayQueue } from '../types'

// Pins the frontend TodayQueue type to the backend OpenAPI contract: a
// field added, renamed or dropped on one side fails here.
const SAMPLE: Record<keyof TodayQueue, number> = {
  to_pick: 0, to_pack: 0, stock_issues: 0, unmatched_skus: 0, expiring_batches: 0, expired_batches: 0, ship_today: 0,
}

function requiredFields(yaml: string, schema: string): string[] {
  const block = yaml.replace(/\r\n/g, '\n').split(`\n    ${schema}:\n`)[1] ?? ''
  const m = /\n {6}required: \[([^\]]*)\]/.exec(`\n${block}`)
  return m ? m[1].split(',').map(s => s.trim()) : []
}

describe('TodayQueue contract', () => {
  it('has exactly the fields the OpenAPI spec requires', () => {
    expect(Object.keys(SAMPLE).sort()).toEqual(requiredFields(dashboardSpec, 'TodayQueue').sort())
  })
})

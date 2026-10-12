import { describe, expect, it } from 'vitest'
import dashboardSpec from '../../../../backend/api/dashboard/openapi.yaml?raw'
import type { SalesComparison, SalesFigures } from '../types'

// Pins the frontend sales comparison types to the backend OpenAPI
// contract: a field added, renamed or dropped on one side fails here.
const FIGURES: Record<keyof SalesFigures, true> = {
  from: true, to: true, revenue_minor_units: true, order_count: true, by_channel: true,
}
const COMPARISON: Record<keyof SalesComparison, true> = {
  period: true, currency_code: true, current: true, previous: true, change_percent: true,
}

function requiredFields(yaml: string, schema: string): string[] {
  const block = yaml.replace(/\r\n/g, '\n').split(`\n    ${schema}:\n`)[1] ?? ''
  const m = /\n {6}required: \[([^\]]*)\]/.exec(`\n${block}`)
  return m ? m[1].split(',').map(s => s.trim()) : []
}

describe('sales comparison contract', () => {
  it('SalesFigures has exactly the fields the OpenAPI spec requires', () => {
    expect(Object.keys(FIGURES).sort()).toEqual(requiredFields(dashboardSpec, 'SalesFigures').sort())
  })

  it('SalesComparison has exactly the fields the OpenAPI spec requires', () => {
    expect(Object.keys(COMPARISON).sort()).toEqual(requiredFields(dashboardSpec, 'SalesComparison').sort())
  })
})

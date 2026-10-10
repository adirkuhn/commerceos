import type { TFunction } from 'i18next'
import type { Insight } from './types'

// insightText is a business insight in the admin's language: its rule
// code's text (reports.json insights.*) filled with its params. Unknown
// rules, and responses without params, show the backend's English message.
export function insightText(t: TFunction<'reports'>, insight: Insight, language: string): string {
  const p = insight.params ?? {}
  const count = typeof p.count === 'number' ? p.count : undefined
  const percent = typeof p.percent === 'number'
    ? new Intl.NumberFormat(language, { maximumFractionDigits: 1 }).format(p.percent)
    : undefined
  switch (insight.code) {
    case 'revenue_declining':
      return percent === undefined ? insight.message : t('insights.revenueDeclining', { percent })
    case 'incomplete_cost_data':
      return percent === undefined ? insight.message : t('insights.incompleteCostData', { percent })
    case 'slow_moving_stock':
      return count === undefined ? insight.message : t('insights.slowMovingStock', { count })
    case 'hidden_gem_products':
      return count === undefined ? insight.message : t('insights.hiddenGemProducts', { count })
    case 'problem_products':
      return count === undefined ? insight.message : t('insights.problemProducts', { count })
    case 'restock_risk':
      return count === undefined ? insight.message : t('insights.restockRisk', { count })
    default:
      return insight.message
  }
}

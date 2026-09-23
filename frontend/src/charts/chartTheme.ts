import type { CSSProperties } from 'react'
import { useAppliedTheme } from '../theme/useAppliedTheme'
import type { ResolvedTheme } from '../theme/themePreference'

// Chart colours. Recharts writes colours into SVG attributes, which can't
// read CSS variables, so charts take hex values resolved for the current
// appearance from here instead of from index.css.
//
// Categorical series: slot 1 is the brand ultramarine, then orange, aqua,
// yellow, magenta from the dataviz reference palette. Its green and red
// slots are left out so no series looks like success/danger. Checked with
// the dataviz validator (validate_palette.js), adjacent pairs:
//   light on #FFFFFF: all gates pass; slots 3-5 are below 3:1 on the
//     surface, so every multi-series chart keeps a legend + tooltip.
//   dark on #1F2937:  all gates pass, all slots >= 3:1.
// Slots are assigned in fixed order and never cycled; a 6th series folds
// into `other`.
export interface ChartTheme {
  series: readonly string[]
  other: string
  grid: string
  axis: string
  label: string
  surface: string
  tooltip: CSSProperties
}

export const CHART_THEMES: Record<ResolvedTheme, ChartTheme> = {
  light: {
    series: ['#4447D9', '#EB6834', '#1BAF7A', '#EDA100', '#E87BA4'],
    other: '#8B93A1',
    grid: '#E5E7EB',
    axis: '#646C79',
    label: '#374151',
    surface: '#FFFFFF',
    tooltip: {
      backgroundColor: '#FFFFFF', border: '1px solid #E5E7EB', borderRadius: '8px',
      fontSize: '12px', color: '#111827', boxShadow: '0 8px 24px rgba(0,0,0,0.12)',
    },
  },
  dark: {
    series: ['#7C80F5', '#D95926', '#199E70', '#C98500', '#D55181'],
    other: '#6B7280',
    grid: '#2D3748',
    axis: '#939CAB',
    label: '#D1D5DB',
    surface: '#1F2937',
    tooltip: {
      backgroundColor: '#111827', border: '1px solid #2D3748', borderRadius: '8px',
      fontSize: '12px', color: '#F3F4F6',
    },
  },
}

export function seriesColor(theme: ChartTheme, index: number): string {
  return theme.series[index] ?? theme.other
}

// Sales channels keep one colour everywhere (Dashboard, reports) and when
// others are filtered out: the colour follows the channel, not its rank.
const CHANNEL_SLOTS: Readonly<Record<string, number>> = {
  woocommerce: 0,
  bol: 1,
  pos: 2,
  internal: 3,
  manual: 4,
}

export function channelSeriesColor(theme: ChartTheme, source: string): string {
  const slot = CHANNEL_SLOTS[source]
  return slot === undefined ? theme.other : seriesColor(theme, slot)
}

export function useChartTheme(): ChartTheme {
  return CHART_THEMES[useAppliedTheme()]
}

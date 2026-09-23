import { describe, it, expect } from 'vitest'
import { CHART_THEMES, channelSeriesColor, seriesColor } from '../chartTheme'

describe('chart themes', () => {
  it('selects a separate, validated step set per appearance', () => {
    expect(CHART_THEMES.light.series).toHaveLength(5)
    expect(CHART_THEMES.dark.series).toHaveLength(5)
    expect(CHART_THEMES.light.series).not.toEqual(CHART_THEMES.dark.series)
  })

  it('leads with the brand ultramarine', () => {
    expect(CHART_THEMES.light.series[0]).toBe('#4447D9')
    expect(CHART_THEMES.dark.series[0]).toBe('#7C80F5')
  })

  it('never uses a status colour as a series colour', () => {
    const status = ['#15803D', '#DC2626', '#B45309', '#4ADE80', '#F87171', '#FBBF24']
    for (const mode of ['light', 'dark'] as const) {
      for (const c of CHART_THEMES[mode].series) expect(status).not.toContain(c.toUpperCase())
    }
  })

  it('keeps axis text readable (>= 4.5:1 tokens) and gridlines recessive', () => {
    expect(CHART_THEMES.light.axis).toBe('#646C79')
    expect(CHART_THEMES.dark.axis).toBe('#939CAB')
    expect(CHART_THEMES.light.grid).toBe('#E5E7EB')
  })
})

describe('seriesColor', () => {
  it('uses slots in fixed order and never cycles past the palette', () => {
    const t = CHART_THEMES.light
    expect(seriesColor(t, 0)).toBe(t.series[0])
    expect(seriesColor(t, 4)).toBe(t.series[4])
    expect(seriesColor(t, 5)).toBe(t.other)
  })
})

describe('channelSeriesColor', () => {
  const t = CHART_THEMES.light

  it('follows the channel, not its position in the list', () => {
    expect(channelSeriesColor(t, 'bol')).toBe(channelSeriesColor(t, 'bol'))
    expect(channelSeriesColor(t, 'woocommerce')).toBe(t.series[0])
    expect(channelSeriesColor(t, 'bol')).toBe(t.series[1])
  })

  it('gives each known channel its own colour', () => {
    const known = ['woocommerce', 'bol', 'pos', 'internal', 'manual'].map(s => channelSeriesColor(t, s))
    expect(new Set(known).size).toBe(5)
  })

  it('folds channels it does not know into a neutral Other colour', () => {
    expect(channelSeriesColor(t, 'something-new')).toBe(t.other)
  })
})

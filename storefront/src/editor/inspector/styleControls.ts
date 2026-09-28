import type { Breakpoint, NodeStyle, StyleProperty } from '../../core/document'
import type { StyleGroup } from '../../components/types'
import type { DeviceId } from '../useGrapesEditor'

// The inspector's style controls: every property of the style vocabulary,
// grouped the way merchants think about them. Options mirror the backend
// contract (a test checks every property is offered exactly once).

export interface StyleControl {
  property: StyleProperty
  label: string
  kind: 'select' | 'color'
  options?: { value: string; label: string }[]
}

const opts = (...values: string[]) => values.map(v => ({ value: v, label: v === 'none' ? 'None' : v }))
const SPACING = opts('none', 'xs', 'sm', 'md', 'lg', 'xl', '2xl', '3xl')

export const STYLE_CONTROLS: Record<StyleGroup, StyleControl[]> = {
  spacing: [
    { property: 'padding_y', label: 'Padding top/bottom', kind: 'select', options: SPACING },
    { property: 'padding_x', label: 'Padding left/right', kind: 'select', options: SPACING },
    { property: 'margin_top', label: 'Space above', kind: 'select', options: SPACING },
    { property: 'margin_bottom', label: 'Space below', kind: 'select', options: SPACING },
    { property: 'gap', label: 'Gap between items', kind: 'select', options: SPACING },
  ],
  typography: [
    { property: 'font_family', label: 'Font', kind: 'select', options: [{ value: 'heading', label: 'Heading font' }, { value: 'body', label: 'Body font' }] },
    { property: 'font_size', label: 'Size', kind: 'select', options: opts('xs', 'sm', 'base', 'lg', 'xl', '2xl', '3xl', '4xl', '5xl') },
    { property: 'font_weight', label: 'Weight', kind: 'select', options: opts('regular', 'medium', 'semibold', 'bold') },
    { property: 'text_align', label: 'Alignment', kind: 'select', options: opts('left', 'center', 'right') },
    { property: 'line_height', label: 'Line height', kind: 'select', options: opts('tight', 'normal', 'relaxed') },
    { property: 'letter_spacing', label: 'Letter spacing', kind: 'select', options: opts('tight', 'normal', 'wide') },
    { property: 'text_transform', label: 'Capitals', kind: 'select', options: [{ value: 'none', label: 'As typed' }, { value: 'uppercase', label: 'UPPERCASE' }] },
  ],
  colors: [
    { property: 'color', label: 'Text colour', kind: 'color' },
    { property: 'background_color', label: 'Background colour', kind: 'color' },
  ],
  border: [
    { property: 'border_width', label: 'Border', kind: 'select', options: opts('none', 'thin', 'medium', 'thick') },
    { property: 'border_color', label: 'Border colour', kind: 'color' },
    { property: 'border_radius', label: 'Corner rounding', kind: 'select', options: opts('none', 'sm', 'md', 'lg', 'xl', 'full') },
    { property: 'shadow', label: 'Shadow', kind: 'select', options: opts('none', 'sm', 'md', 'lg') },
  ],
  background: [
    { property: 'background_image', label: 'Background image', kind: 'select' },
    { property: 'background_size', label: 'Image fit', kind: 'select', options: opts('cover', 'contain') },
    { property: 'background_position', label: 'Image position', kind: 'select', options: opts('center', 'top', 'bottom') },
  ],
  layout: [
    { property: 'width', label: 'Width', kind: 'select', options: [{ value: 'auto', label: 'Auto' }, { value: 'full', label: 'Full' }, { value: 'content', label: 'Page width' }, { value: 'narrow', label: 'Narrow' }] },
    { property: 'min_height', label: 'Minimum height', kind: 'select', options: opts('none', 'sm', 'md', 'lg', 'screen') },
    { property: 'aspect_ratio', label: 'Shape', kind: 'select', options: opts('auto', '1:1', '4:3', '3:2', '16:9', '3:4', '2:3') },
    { property: 'align_items', label: 'Align items', kind: 'select', options: opts('start', 'center', 'end', 'stretch') },
    { property: 'justify_content', label: 'Distribute items', kind: 'select', options: opts('start', 'center', 'end', 'between') },
    { property: 'columns', label: 'Grid columns', kind: 'select', options: opts('1', '2', '3', '4', '5', '6') },
  ],
  visibility: [
    { property: 'display', label: 'Visible', kind: 'select', options: [{ value: 'show', label: 'Show' }, { value: 'hide', label: 'Hide' }] },
  ],
}

export const GROUP_LABELS: Record<StyleGroup, string> = {
  spacing: 'Spacing', typography: 'Typography', colors: 'Colours', border: 'Border', background: 'Background', layout: 'Layout', visibility: 'Visibility',
}

export function controlsFor(groups: StyleGroup[]) {
  return groups.map(group => ({ group, label: GROUP_LABELS[group], controls: STYLE_CONTROLS[group] }))
}

export function breakpointFor(device: DeviceId): Breakpoint {
  return device === 'desktop' ? 'base' : device
}

/** A new style with one value set ('' clears it); empty breakpoints are dropped. */
export function setStyleValue(style: NodeStyle, bp: Breakpoint, property: StyleProperty, value: string): NodeStyle {
  const next: NodeStyle = { ...style, [bp]: { ...(style[bp] ?? {}) } }
  if (value) next[bp]![property] = value
  else delete next[bp]![property]
  if (!Object.keys(next[bp]!).length) delete next[bp]
  return next
}

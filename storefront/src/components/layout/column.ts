import type { ComponentDefinition } from '../types'

export interface ColumnProps {
  span: number
}

export const column: ComponentDefinition<ColumnProps> = {
  type: 'column',
  name: 'Column',
  category: 'layout',
  icon: 'column',
  description: 'One column. Give it a bigger span to make it wider than its neighbours.',
  props: { span: { kind: 'select', label: 'Width', options: [1, 2, 3, 4].map(n => ({ value: n, label: `${n}×` })) } },
  defaults: { span: 1 },
  container: true,
  element: p => ({ tag: 'div', className: `cos-column cos-span-${Math.min(Math.max(Number(p.span) || 1, 1), 4)}` }),
  allowedChildren: [],
  allowedParents: ['columns'],
  dataContext: '',
  requiredIn: [],
  styleGroups: ['spacing', 'colors', 'background', 'border', 'layout', 'visibility'],
  editor: { removable: true, copyable: true, draggable: true },
}

import type { ComponentDefinition } from '../types'

export interface SectionProps {
  full_width: boolean
}

export const section: ComponentDefinition<SectionProps> = {
  type: 'section',
  name: 'Section',
  category: 'layout',
  icon: 'rows',
  description: 'A full-width band of the page. Everything on a page lives inside a section.',
  props: { full_width: { kind: 'toggle', label: 'Edge to edge' } },
  defaults: { full_width: false },
  container: true,
  element: p => ({ tag: 'section', className: p.full_width ? 'cos-section' : 'cos-section cos-contained' }),
  allowedChildren: [],
  allowedParents: ['root'],
  dataContext: '',
  requiredIn: [],
  styleGroups: ['spacing', 'colors', 'background', 'border', 'layout', 'visibility'],
  editor: { removable: true, copyable: true, draggable: true },
}

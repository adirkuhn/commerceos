import type { ComponentDefinition } from '../types'

export interface HeadingProps {
  text: string
  level: number
}

export const heading: ComponentDefinition<HeadingProps> = {
  type: 'heading',
  name: 'Heading',
  category: 'content',
  icon: 'heading',
  description: 'A title. Level 1 once per page; use 2-3 for sections.',
  props: {
    text: { kind: 'text', label: 'Text', maxLength: 300 },
    level: {
      kind: 'select', label: 'Level',
      options: [1, 2, 3, 4, 5, 6].map(l => ({ value: l, label: `H${l}` })),
    },
  },
  defaults: { text: 'Heading', level: 2 },
  container: false,
  allowedChildren: [],
  allowedParents: [],
  dataContext: '',
  requiredIn: [],
  styleGroups: ['typography', 'colors', 'spacing', 'visibility'],
  editor: { removable: true, copyable: true, draggable: true },
}

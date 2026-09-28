import type { ComponentDefinition } from '../types'

export interface ColumnsProps {
  stack_on: 'tablet' | 'mobile' | 'never'
}

export const columns: ComponentDefinition<ColumnsProps> = {
  type: 'columns',
  name: 'Columns',
  category: 'layout',
  icon: 'columns',
  description: 'Side-by-side columns that stack on smaller screens.',
  props: {
    stack_on: {
      kind: 'select', label: 'Stack columns on',
      options: [{ value: 'tablet', label: 'Tablet and mobile' }, { value: 'mobile', label: 'Mobile only' }, { value: 'never', label: 'Never' }],
    },
  },
  defaults: { stack_on: 'mobile' },
  container: true,
  element: p => ({ tag: 'div', className: `cos-columns cos-stack-${p.stack_on ?? 'mobile'}` }),
  allowedChildren: ['column'],
  allowedParents: [],
  dataContext: '',
  requiredIn: [],
  styleGroups: ['spacing', 'colors', 'background', 'layout', 'visibility'],
  editor: { removable: true, copyable: true, draggable: true },
}

import type { ComponentDefinition } from '../types'

export interface ButtonProps {
  label: string
  link: string
  variant: 'primary' | 'secondary' | 'outline'
  new_tab: boolean
}

export const button: ComponentDefinition<ButtonProps> = {
  type: 'button',
  name: 'Button',
  category: 'content',
  icon: 'button',
  description: 'A call-to-action link styled as a button.',
  props: {
    label: { kind: 'text', label: 'Label', maxLength: 80 },
    link: { kind: 'link', label: 'Link' },
    variant: {
      kind: 'select', label: 'Style',
      options: [{ value: 'primary', label: 'Primary' }, { value: 'secondary', label: 'Secondary' }, { value: 'outline', label: 'Outline' }],
    },
    new_tab: { kind: 'toggle', label: 'Open in a new tab' },
  },
  defaults: { label: 'Shop now', link: '/', variant: 'primary', new_tab: false },
  container: false,
  allowedChildren: [],
  allowedParents: [],
  dataContext: '',
  requiredIn: [],
  styleGroups: ['spacing', 'layout', 'visibility'],
  editor: { removable: true, copyable: true, draggable: true },
}

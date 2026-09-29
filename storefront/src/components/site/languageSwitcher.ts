import type { ComponentDefinition } from '../types'

export interface LanguageSwitcherProps { display: 'names' | 'codes' }

export const languageSwitcher: ComponentDefinition<LanguageSwitcherProps> = {
  type: 'language-switcher', name: 'Language switcher', category: 'navigation', icon: 'globe',
  description: 'Lets shoppers switch between the languages your store offers.',
  props: {
    display: { kind: 'select', label: 'Show', options: [{ value: 'names', label: 'Language names' }, { value: 'codes', label: 'Short codes (EN, PT)' }] },
  },
  defaults: { display: 'names' },
  container: false, allowedChildren: [], allowedParents: [], dataContext: '', requiredIn: [],
  editor: { removable: true, copyable: true, draggable: true },
  styleGroups: ['typography', 'spacing', 'visibility'],
}

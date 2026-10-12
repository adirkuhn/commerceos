import type { ElementType, RefObject } from 'react'
import { useTranslation } from 'react-i18next'
import { AddIcon, LayersIcon } from './icons'

export type SidePanelTab = 'add' | 'layers'

interface Props {
  tab: SidePanelTab
  onTab: (tab: SidePanelTab) => void
  // GrapesJS renders the blocks and layers into these once, so both stay
  // mounted and the closed one is only hidden.
  blocksRef: RefObject<HTMLDivElement>
  layersRef: RefObject<HTMLDivElement>
}

const TABS: Array<{ id: SidePanelTab; icon: ElementType; label: 'layout.add' | 'layout.layers' }> = [
  { id: 'add', icon: AddIcon, label: 'layout.add' },
  { id: 'layers', icon: LayersIcon, label: 'layout.layers' },
]

/** Left panel: the components to add, or the page's layers. */
export function EditorSidePanel({ tab, onTab, blocksRef, layersRef }: Props) {
  const { t } = useTranslation('editor')
  return (
    <aside className="cos-editor-panel flex w-64 shrink-0 flex-col border-r border-ed-line bg-ed-raised" aria-label={t('layout.componentsAndLayers')}>
      <div role="tablist" className="grid shrink-0 grid-cols-2 gap-1 border-b border-ed-line p-1.5">
        {TABS.map(({ id, icon: Icon, label }) => (
          <button key={id} type="button" role="tab" id={`editor-tab-${id}`} aria-selected={tab === id} aria-controls={`editor-panel-${id}`}
            onClick={() => onTab(id)}
            className={`flex items-center justify-center gap-2 rounded-ed py-1.5 text-sm font-medium transition-colors ${
              tab === id ? 'bg-ed-accent/10 text-ed-accent' : 'text-ed-ink-3 hover:bg-ed-hover hover:text-ed-ink'}`}>
            <Icon size={16} />
            {t(label)}
          </button>
        ))}
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto">
        <div ref={blocksRef} id="editor-panel-add" role="tabpanel" aria-labelledby="editor-tab-add" hidden={tab !== 'add'} data-testid="editor-blocks" />
        <div ref={layersRef} id="editor-panel-layers" role="tabpanel" aria-labelledby="editor-tab-layers" hidden={tab !== 'layers'} data-testid="editor-layers" />
      </div>
    </aside>
  )
}

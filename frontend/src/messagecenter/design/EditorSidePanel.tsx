import type { ElementType, ReactNode, RefObject } from 'react'
import { useTranslation } from 'react-i18next'
import { Braces, Layers, LayoutGrid, Paintbrush } from 'lucide-react'
import clsx from 'clsx'

export type SidePanelTab = 'blocks' | 'style' | 'layers' | 'variables'

interface Props {
  tab: SidePanelTab
  onTab: (tab: SidePanelTab) => void
  // GrapesJS renders into these at start-up, so they stay mounted and are
  // only hidden while another tab is open.
  refs: { blocks: RefObject<HTMLDivElement>; styles: RefObject<HTMLDivElement>; layers: RefObject<HTMLDivElement> }
  variables: ReactNode
}

const TABS: Array<{ id: SidePanelTab; icon: ElementType }> = [
  { id: 'blocks', icon: LayoutGrid },
  { id: 'style', icon: Paintbrush },
  { id: 'layers', icon: Layers },
  { id: 'variables', icon: Braces },
]

// EditorSidePanel is the email editor's one side panel: what to add
// (blocks), how the selected block looks (style), the email's structure
// (layers) and the trigger's variables, one tab each.
export function EditorSidePanel({ tab, onTab, refs, variables }: Props) {
  const { t } = useTranslation('messages')
  return (
    <aside className="cos-email-panel w-72 shrink-0 flex flex-col border-l border-line bg-raised min-h-0">
      <div role="tablist" className="grid grid-cols-4 gap-1 p-1.5 border-b border-line shrink-0">
        {TABS.map(({ id, icon: Icon }) => (
          <button
            key={id}
            role="tab"
            id={`email-tab-${id}`}
            aria-selected={tab === id}
            aria-controls={`email-panel-${id}`}
            onClick={() => onTab(id)}
            className={clsx(
              'flex flex-col items-center gap-1 py-1.5 rounded-lg text-2xs font-medium transition-colors',
              tab === id ? 'bg-accent/10 text-accent' : 'text-ink-3 hover:bg-hover hover:text-ink',
            )}
          >
            <Icon size={16} aria-hidden />
            {t(`design.tabs.${id}`)}
          </button>
        ))}
      </div>
      <div className="flex-1 min-h-0 overflow-y-auto">
        <div ref={refs.blocks} id="email-panel-blocks" role="tabpanel" aria-labelledby="email-tab-blocks" hidden={tab !== 'blocks'} data-testid="panel-blocks" />
        <div ref={refs.styles} id="email-panel-style" role="tabpanel" aria-labelledby="email-tab-style" hidden={tab !== 'style'} data-testid="panel-styles" />
        <div ref={refs.layers} id="email-panel-layers" role="tabpanel" aria-labelledby="email-tab-layers" hidden={tab !== 'layers'} data-testid="panel-layers" />
        <div id="email-panel-variables" role="tabpanel" aria-labelledby="email-tab-variables" hidden={tab !== 'variables'}>
          {variables}
        </div>
      </div>
    </aside>
  )
}

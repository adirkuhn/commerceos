import { Plug } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'

export const EDITOR_TABS = ['general', 'woocommerce', 'bol'] as const

export type EditorTab = typeof EDITOR_TABS[number]

// General holds the shared data every provider needs; each channel tab
// maps the same product to that channel's own fields and is only offered
// while that channel is connected.
const CHANNEL_TAB: Record<EditorTab, boolean> = { general: false, woocommerce: true, bol: true }

interface ProductEditorTabsProps {
  tabs: EditorTab[]
  active: EditorTab
  onSelect: (tab: EditorTab) => void
}

export function ProductEditorTabs({ tabs, active, onSelect }: ProductEditorTabsProps) {
  const { t } = useTranslation('pms')
  const labels: Record<EditorTab, string> = { general: t('tabs.general'), woocommerce: t('tabs.woocommerce'), bol: t('tabs.bol') }
  return (
    <div className="flex items-center gap-1 px-5 border-b border-line bg-surface shrink-0">
      {tabs.map(tab => {
        const label = labels[tab]
        const channel = CHANNEL_TAB[tab]
        return (
          <button key={tab} aria-pressed={active === tab}
            onClick={() => onSelect(tab)}
            className={clsx(
              channel
                ? 'flex items-center gap-1.5 px-3 py-2.5 text-xs font-medium border-b-2 -mb-px transition-colors'
                : 'px-3 py-2.5 text-xs font-medium border-b-2 -mb-px transition-colors',
              active === tab
                ? 'border-accent text-accent'
                : 'border-transparent text-ink-4 hover:text-ink-2',
            )}
          >
            {channel ? <><Plug size={12} />{` ${label}`}</> : label}
          </button>
        )
      })}
    </div>
  )
}

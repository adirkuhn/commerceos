import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { useQueryEnum } from '../routing/useQueryState'
import { useSuppliers } from '../suppliers/hooks/useSuppliers'
import { ReorderPlanTab } from '../purchaseplanning/ReorderPlanTab'
import { SeasonPlanTab } from '../purchaseplanning/SeasonPlanTab'

const TABS = ['reorder', 'season'] as const

export default function PurchasePlanner() {
  const { token } = useAuth()
  const { t } = useTranslation('purchasePlanning')
  const { suppliers } = useSuppliers(token ?? '')
  // ?tab=season; each tab keeps its own window in the URL too.
  const [tab, setTab] = useQueryEnum('tab', TABS, 'reorder')
  const labels = { reorder: t('tabs.reorder'), season: t('tabs.season') }

  return (
    <div className="flex flex-col h-full">
      <div role="tablist" className="flex items-center gap-1 px-5 pt-3 bg-sunken shrink-0">
        {TABS.map(id => (
          <button
            key={id}
            role="tab"
            aria-selected={tab === id}
            onClick={() => setTab(id)}
            className={clsx(
              'px-3 py-1.5 rounded-lg text-xs font-medium transition-colors',
              tab === id ? 'dark:bg-night-800 bg-gray-100 text-ink' : 'text-ink-4 hover:text-ink-2',
            )}
          >
            {labels[id]}
          </button>
        ))}
      </div>
      {tab === 'season'
        ? <SeasonPlanTab token={token ?? ''} suppliers={suppliers} />
        : <ReorderPlanTab token={token ?? ''} suppliers={suppliers} />}
    </div>
  )
}

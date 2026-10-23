import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { useQueryEnum, useQueryString } from '../routing/useQueryState'
import { parseDiscount } from '../pricelist/pricing'
import { DiscountField } from '../pricelist/DiscountField'
import { PriceListTab } from '../pricelist/PriceListTab'
import { QuoteTab } from '../pricelist/QuoteTab'
import { SavedQuotesTab } from '../pricelist/SavedQuotesTab'

const TABS = ['list', 'quote', 'saved'] as const

// Price list with a what-if discount, and a B2B quote built on the same
// discount: ?tab=quote&discount=30 (the quote's lines are in ?lines=), and
// the saved quotes (?tab=saved).
export default function PriceList() {
  const { token } = useAuth()
  const { t } = useTranslation('priceList')
  const [tab, setTab] = useQueryEnum('tab', TABS, 'list')
  const [discountRaw, setDiscount] = useQueryString('discount')
  const discount = parseDiscount(discountRaw)

  return (
    <div className="flex flex-col h-full">
      <div className="flex flex-wrap items-center gap-3 px-5 pt-3 pb-2 bg-sunken shrink-0">
        <div role="tablist" className="flex items-center gap-1">
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
              {t(`tabs.${id}`)}
            </button>
          ))}
        </div>
        {tab !== 'saved' && (
          <div className="ml-auto">
            <DiscountField value={discountRaw ?? ''} onChange={setDiscount} />
          </div>
        )}
      </div>
      {tab === 'quote' && <QuoteTab token={token ?? ''} discount={discount} />}
      {tab === 'saved' && <SavedQuotesTab token={token ?? ''} />}
      {tab === 'list' && <PriceListTab token={token ?? ''} discount={discount} />}
    </div>
  )
}

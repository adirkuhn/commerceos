import { PackageCheck } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { MissingPickItem } from './types'
import { MissingLineCard } from './ScanSessionLines'

interface PickingDoneScreenProps {
  missingItems?: MissingPickItem[]
  error: string
  startingPacking: boolean
  onContinueToPacking: () => void
  onDone: () => void
}

// "Order picked": what wasn't fully picked, then on to packing or Done.
export function PickingDoneScreen({ missingItems, error, startingPacking, onContinueToPacking, onDone }: PickingDoneScreenProps) {
  const { t } = useTranslation('scanner')
  return (
    <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
      <div className="flex flex-col items-center text-center gap-3 px-6 pt-8 pb-4 shrink-0">
        <div className="w-14 h-14 rounded-2xl bg-green-600/20 flex items-center justify-center">
          <PackageCheck size={26} className="text-green-400" />
        </div>
        <h1 className="text-lg font-semibold">{t('pickDone.title')}</h1>
        {!missingItems?.length && <p className="text-sm text-gray-400">{t('pickDone.all')}</p>}
      </div>
      {!!missingItems?.length && (
        <div className="flex-1 overflow-y-auto px-6 pb-4 space-y-2">
          <p className="text-sm text-amber-400">{t('pickDone.notFully', { count: missingItems.length })}</p>
          {missingItems.map(m => (
            <MissingLineCard key={m.line_item_id} activity="picking" name={m.name} expectedQty={m.expected_qty} actualQty={m.picked_qty} />
          ))}
        </div>
      )}
      <div className="p-6 shrink-0 space-y-2">
        {error && <p className="text-sm text-red-400">{error}</p>}
        <button
          onClick={onContinueToPacking} disabled={startingPacking}
          className="w-full py-3 rounded-xl bg-accent-solid text-white font-semibold hover:bg-accent-solid-hover disabled:opacity-50"
        >
          {startingPacking ? t('pickDone.starting') : t('pickDone.continue')}
        </button>
        <button onClick={onDone} className="w-full py-2.5 rounded-xl bg-night-800 text-gray-300 text-sm font-medium">
          {t('shared.done')}
        </button>
      </div>
    </div>
  )
}

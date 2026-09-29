import type { ReactNode } from 'react'
import { ArrowLeft } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { ResourceState } from '../routing/useResource'

interface ScannerLoadGateProps<T> {
  resource: ResourceState<T>
  onBack: () => void
  children: (data: T) => ReactNode
}

// Full-screen loading/error shell for a scanner screen reached by URL. On
// error it always offers a way back -- a stale or mistyped deep link must
// never strand the operator on a dead screen.
export function ScannerLoadGate<T>({ resource, onBack, children }: ScannerLoadGateProps<T>) {
  const { t } = useTranslation(['scanner', 'common'])
  if (resource.data !== null) return <>{children(resource.data)}</>

  return (
    <div className="fixed inset-0 flex flex-col items-center justify-center gap-4 p-6 bg-night-950 text-gray-100 text-center">
      {resource.loading ? (
        <p className="text-sm text-gray-400">{t('common:status.loading')}</p>
      ) : (
        <>
          <p className="text-sm text-red-400">{resource.error || t('shared.notFound')}</p>
          <button
            onClick={onBack}
            className="flex items-center gap-1.5 px-4 py-3 rounded-xl bg-night-800 border border-night-700 text-gray-200 font-medium"
          >
            <ArrowLeft size={16} /> {t('shared.back')}
          </button>
        </>
      )}
    </div>
  )
}

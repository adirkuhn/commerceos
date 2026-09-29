import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import { BrandMark } from '../components/brand/BrandMark'
import { Boxes, PackageCheck, Tag, ClipboardList, LogOut, Trash2, Package } from 'lucide-react'

export type ScannerAppID = 'receive-po' | 'inventory' | 'price-lookup' | 'pick-order' | 'pack-order' | 'discard'

interface AppLauncherProps {
  onSelect: (app: ScannerAppID) => void
  onLogout: () => void
}

const apps = (t: TFunction<'scanner'>): { id: ScannerAppID; label: string; icon: typeof PackageCheck; enabled: boolean }[] => [
  { id: 'receive-po', label: t('apps.receivePo'), icon: PackageCheck, enabled: true },
  { id: 'inventory', label: t('apps.inventory'), icon: Boxes, enabled: true },
  { id: 'price-lookup', label: t('apps.priceLookup'), icon: Tag, enabled: true },
  { id: 'discard', label: t('apps.discard'), icon: Trash2, enabled: true },
  { id: 'pick-order', label: t('apps.pickOrder'), icon: ClipboardList, enabled: true },
  { id: 'pack-order', label: t('apps.packOrder'), icon: Package, enabled: true },
]

// Entry point once a handheld is paired -- hosts every scanner mini-app.
// Large touch targets throughout: this screen (and every app it launches)
// must work by tapping alone, not just by physical trigger-scanning.
export function AppLauncher({ onSelect, onLogout }: AppLauncherProps) {
  const { t } = useTranslation('scanner')
  return (
    <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-night-700 shrink-0">
        <span className="w-8 h-8 rounded-lg bg-accent-solid flex items-center justify-center shrink-0">
          <BrandMark size={18} className="text-white" />
        </span>
        <h1 className="flex-1 text-base font-semibold">{t('apps.title')}</h1>
        <button aria-label={t('shared.unpair')} onClick={onLogout} className="w-11 h-11 -my-2 -mr-2 flex items-center justify-center rounded-lg hover:bg-night-800" title={t('shared.unpair')}>
          <LogOut size={16} className="text-gray-400" />
        </button>
      </div>
      <div className="flex-1 overflow-y-auto p-4 grid grid-cols-2 gap-3">
        {apps(t).map(app => (
          <button
            key={app.id}
            onClick={() => app.enabled && onSelect(app.id)}
            disabled={!app.enabled}
            className="flex flex-col items-center justify-center gap-3 aspect-square rounded-2xl bg-night-800 border border-night-700 disabled:opacity-40 hover:border-accent disabled:hover:border-night-700 transition-colors"
          >
            <div className="w-12 h-12 rounded-xl bg-night-700 flex items-center justify-center">
              <app.icon size={22} className="text-gray-300" />
            </div>
            <span className="text-sm font-medium">{app.label}</span>
            {!app.enabled && <span className="text-2xs text-gray-400">{t('apps.comingSoon')}</span>}
          </button>
        ))}
      </div>
    </div>
  )
}

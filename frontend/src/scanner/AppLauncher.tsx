import { Boxes, PackageCheck, Tag, ClipboardList, LogOut } from 'lucide-react'

export type ScannerAppID = 'receive-po' | 'inventory' | 'price-lookup' | 'pick-order'

interface AppLauncherProps {
  onSelect: (app: ScannerAppID) => void
  onLogout: () => void
}

const APPS: { id: ScannerAppID; label: string; icon: typeof PackageCheck; enabled: boolean }[] = [
  { id: 'receive-po', label: 'Receive PO', icon: PackageCheck, enabled: true },
  { id: 'inventory', label: 'Inventory', icon: Boxes, enabled: true },
  { id: 'price-lookup', label: 'Price Lookup', icon: Tag, enabled: true },
  { id: 'pick-order', label: 'Pick Order', icon: ClipboardList, enabled: false },
]

// Entry point once a handheld is paired -- hosts every scanner mini-app.
// Large touch targets throughout: this screen (and every app it launches)
// must work by tapping alone, not just by physical trigger-scanning.
export function AppLauncher({ onSelect, onLogout }: AppLauncherProps) {
  return (
    <div className="fixed inset-0 flex flex-col bg-[#0B1220] text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-[#2D3748] shrink-0">
        <h1 className="flex-1 text-base font-semibold">CommerceOS Scanner</h1>
        <button onClick={onLogout} className="p-2 rounded-lg hover:bg-[#1F2937]" title="Unpair">
          <LogOut size={16} className="text-gray-400" />
        </button>
      </div>
      <div className="flex-1 overflow-y-auto p-4 grid grid-cols-2 gap-3">
        {APPS.map(app => (
          <button
            key={app.id}
            onClick={() => app.enabled && onSelect(app.id)}
            disabled={!app.enabled}
            className="flex flex-col items-center justify-center gap-3 aspect-square rounded-2xl bg-[#1F2937] border border-[#2D3748] disabled:opacity-40 hover:border-blue-500 disabled:hover:border-[#2D3748] transition-colors"
          >
            <div className="w-12 h-12 rounded-xl bg-[#2D3748] flex items-center justify-center">
              <app.icon size={22} className="text-gray-300" />
            </div>
            <span className="text-sm font-medium">{app.label}</span>
            {!app.enabled && <span className="text-[10px] text-gray-500">Coming soon</span>}
          </button>
        ))}
      </div>
    </div>
  )
}

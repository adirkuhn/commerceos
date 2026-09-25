import { X } from 'lucide-react'
import { Dialog } from '../components/common/Dialog'
import type { Connection } from './types'
import type { Warehouse } from '../inventory/types'
import { CURRENCIES } from './integrationsPageHelpers'
import { useImportProducts } from './hooks/useImportProducts'

interface Props {
  token: string
  connection: Connection
  warehouses: Warehouse[]
  tenantDefaultCurrency: string | undefined
  onClose: () => void
}

// "Import products" modal for a WooCommerce connection: pick warehouse +
// currency, start a background import job.
export function ImportProductsDialog({ token, connection, warehouses, tenantDefaultCurrency, onClose }: Props) {
  const imp = useImportProducts(token, connection, tenantDefaultCurrency)

  return (
    <Dialog onClose={onClose} className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={onClose} />
      <div className="relative w-full max-w-sm bg-raised rounded-xl shadow-2xl border border-line p-5">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-semibold text-ink">Import products</h3>
          <button onClick={onClose} aria-label="Close">
            <X size={14} className="text-ink-4" />
          </button>
        </div>

        {imp.job ? (
          <div className="space-y-3">
            <p className="text-xs text-ink-2">
              Import started — it's running in the background. You'll see a notification when it's done (check the bell icon, top right).
            </p>
            <button
              onClick={onClose}
              className="w-full py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover transition-colors"
            >
              Done
            </button>
          </div>
        ) : (
          <form onSubmit={imp.submit} className="space-y-3">
            <p className="text-2xs text-ink-4">
              Imports every WooCommerce product not yet linked, along with its category. Existing categories are matched by name; missing ones are created.
            </p>
            <div>
              <label htmlFor="import-warehouse-id" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                Default warehouse ID
              </label>
              <select
                id="import-warehouse-id"
                required
                autoFocus
                value={imp.warehouseId}
                onChange={e => imp.setWarehouseId(e.target.value)}
                className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
              >
                <option value="" disabled>Select a warehouse…</option>
                {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
              </select>
              <p className="mt-1.5 text-2xs text-ink-4">
                Imported stock quantities are attributed to this warehouse — WooCommerce has no warehouse concept.
              </p>
            </div>
            <div>
              <label htmlFor="import-currency" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                Default currency
              </label>
              <select
                id="import-currency"
                required
                value={imp.currency}
                onChange={e => imp.setCurrency(e.target.value)}
                className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
              >
                <option value="" disabled>Select a currency…</option>
                {CURRENCIES.map(c => <option key={c} value={c}>{c}</option>)}
              </select>
            </div>
            {imp.error && (
              <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{imp.error}</p>
            )}
            <button
              type="submit"
              disabled={imp.importing}
              className="w-full py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
            >
              {imp.importing ? 'Importing…' : 'Start import'}
            </button>
          </form>
        )}
      </div>
    </Dialog>
  )
}

import { Trans, useTranslation } from 'react-i18next'
import { X } from 'lucide-react'
import { Dialog } from '../components/common/Dialog'
import type { Connection, DeleteAllImportedProductsResult } from './types'
import { useDeleteImportedProducts } from './hooks/useDeleteImportedProducts'

interface Props {
  token: string
  connection: Connection
  onClose: () => void
}

function DeleteSummary({ result, onClose }: { result: DeleteAllImportedProductsResult; onClose: () => void }) {
  const { t } = useTranslation(['integrations', 'common'])
  const errors = result.errors ?? []
  return (
    <div className="space-y-3">
      <p className="text-xs text-ink-2">
        {t('deleteImported.deleted')}{result.deleted_sales_orders > 0 && (
          <> <Trans t={t} i18nKey="deleteImported.salesOrders" count={result.deleted_sales_orders} components={{ strong: <strong /> }} />,</>
        )} <Trans t={t} i18nKey="deleteImported.products" count={result.deleted_products} components={{ strong: <strong /> }} />{' '}
        {t('deleteImported.and')}{' '}
        <Trans t={t} i18nKey="deleteImported.categories" count={result.deleted_categories} components={{ strong: <strong /> }} />.
      </p>
      {errors.length > 0 && (
        <div className="max-h-32 overflow-y-auto space-y-1">
          {errors.map((e, i) => (
            <p key={i} className="text-2xs text-danger bg-red-50 dark:bg-red-500/10 rounded px-2 py-1">{e}</p>
          ))}
        </div>
      )}
      <button
        onClick={onClose}
        className="w-full py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover transition-colors"
      >
        {t('done')}
      </button>
    </div>
  )
}

// "Delete imported products" modal: typed-DELETE confirmation, then a
// summary of what was removed. Only closable by backdrop/X once done.
export function DeleteImportedProductsDialog({ token, connection, onClose }: Props) {
  const { t } = useTranslation(['integrations', 'common'])
  const del = useDeleteImportedProducts(token, connection)

  return (
    <Dialog onClose={onClose} className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={del.result ? onClose : undefined} />
      <div className="relative w-full max-w-sm bg-raised rounded-xl shadow-2xl border border-line p-5">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-semibold text-ink">{t('deleteImported.title')}</h3>
          {del.result && (
            <button onClick={onClose} aria-label={t('common:actions.close')}>
              <X size={14} className="text-ink-4" />
            </button>
          )}
        </div>

        {del.result ? <DeleteSummary result={del.result} onClose={onClose} /> : (
          <form onSubmit={del.submit} className="space-y-3">
            <div className="dark:bg-red-500/10 bg-red-50 border dark:border-red-500/20 border-red-200 rounded-lg p-3">
              <p className="text-xs text-danger">
                <Trans t={t} i18nKey="deleteImported.warning" values={{ store: connection.store_url }} components={{ strong: <strong /> }} />
              </p>
            </div>
            <div>
              <label htmlFor="delete-confirm" className="block text-2xs text-ink-4 caps-label mb-1.5">
                {t('deleteImported.typeToConfirm')}
              </label>
              <input
                id="delete-confirm"
                type="text"
                autoFocus
                autoComplete="off"
                value={del.confirmText}
                onChange={e => del.setConfirmText(e.target.value)}
                className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-red-500"
              />
            </div>
            {del.error && (
              <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{del.error}</p>
            )}
            <div className="grid grid-cols-2 gap-2">
              <button
                type="button"
                onClick={onClose}
                className="py-2 rounded-lg bg-fill text-xs font-medium text-ink-2 transition-colors"
              >
                {t('common:actions.cancel')}
              </button>
              <button
                type="submit"
                disabled={del.deleting || !del.confirmed}
                className="py-2 rounded-lg bg-red-600 text-white text-xs font-medium hover:bg-red-700 disabled:opacity-50 transition-colors"
              >
                {del.deleting ? t('deleteImported.deleting') : t('common:actions.delete')}
              </button>
            </div>
          </form>
        )}
      </div>
    </Dialog>
  )
}

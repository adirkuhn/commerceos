
import { Route, Routes } from 'react-router-dom'
import { SkuRoute } from './SkuRoute'
import { scannerPaths } from './scannerRoutes'
import { useStackNavigation } from '../routing/useStackNavigation'
import { InventoryScanScreen } from './InventoryScanScreens'
import { InventorySKUScreen } from './InventorySKUScreen'

interface InventoryAppProps {
  token: string
  onExit: () => void
}

// Scanner mini-app: scan/type a code to resolve a SKU, then adjust, move, or
// write off its stock -- every action also works by tapping alone (numeric
// keypad inputs + steppers + confirm buttons), not scan-only.
// Mounted at /scanner/inventory/* -- the scan screen is the index route and
// each SKU screen is /scanner/inventory/:code (see ./scannerRoutes), so a
// refresh or Back keeps the operator on the SKU they were working on.
export function InventoryApp({ token, onExit }: InventoryAppProps) {
  const { open } = useStackNavigation()

  return (
    <Routes>
      <Route
        index
        element={<InventoryScanScreen token={token} onResolved={r => open(scannerPaths.inventorySku(r.sku.sku_code))} onExit={onExit} />}
      />
      <Route
        path=":code"
        element={
          <SkuRoute token={token} backTo={scannerPaths.inventory}>
            {(resolved, { update, back }) => (
              <InventorySKUScreen
                token={token}
                sku={resolved.sku}
                productName={resolved.productName}
                onSKUUpdated={updatedSKU => update({ ...resolved, sku: updatedSKU })}
                onBack={back}
              />
            )}
          </SkuRoute>
        }
      />
    </Routes>
  )
}

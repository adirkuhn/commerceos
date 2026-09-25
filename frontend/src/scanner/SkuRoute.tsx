import type { ReactNode } from 'react'
import { useParams } from 'react-router-dom'
import * as pmsApi from '../pms/api'
import { useResource } from '../routing/useResource'
import { useStackNavigation } from '../routing/useStackNavigation'
import { ScannerLoadGate } from './ScannerLoadGate'
import type { Resolved } from './InventoryScanScreens'

interface SkuRouteProps {
  token: string
  // Where "back" lands when this screen wasn't opened from inside the app
  // (deep link / refresh) -- the owning mini-app's scan screen.
  backTo: string
  children: (resolved: Resolved, actions: { update: (next: Resolved) => void; back: () => void }) => ReactNode
}

// Shared ":code" screen for the SKU-centric scanner apps (Inventory, Price
// Lookup, Discard). The URL carries the SKU code -- resolveBarcode accepts
// SKU codes as well as barcodes/aliases -- and the SKU is always re-resolved
// from it, so a refresh shows current data rather than whatever was in
// memory when the code was first scanned.
export function SkuRoute({ token, backTo, children }: SkuRouteProps) {
  const { code = '' } = useParams()
  const { exitTo } = useStackNavigation()
  const resolved = useResource(async (): Promise<Resolved> => {
    const { sku, product_name } = await pmsApi.resolveBarcode(token, code)
    return { sku, productName: product_name }
  }, [token, code])
  const back = () => exitTo(backTo)

  return (
    <ScannerLoadGate resource={resolved} onBack={back}>
      {data => children(data, { update: resolved.setData, back })}
    </ScannerLoadGate>
  )
}

import * as shippingApi from './api'

// Opens a shipment's label in a real new browser tab -- the browser's own
// PDF viewer takes it over completely, so printing from there only ever
// prints the label itself, never our app shell (the old in-app <iframe>
// overlay had window.print() print the whole SPA page around the embedded
// PDF, since the browser has no way to know only the iframe's content was
// meant to be printed). ZPL (thermal-format) labels aren't visually
// renderable, so they download instead of opening a tab.
//
// `tab` must come from a window.open() call made synchronously inside the
// click handler that triggered this, before any `await` -- opening it
// AFTER creating the shipment (or any other async step) runs outside the
// browser's "user activation" window and gets blocked as an unrequested
// popup. Callers with nothing to await first can use openShipmentLabel
// below instead, which opens the tab itself.
export async function loadShipmentLabelIntoTab(
  tab: Window | null, token: string, shipmentId: string, size?: 'A4' | 'A6',
): Promise<Window | null> {
  try {
    const blob = await shippingApi.getShipmentLabel(token, shipmentId, size)
    const url = URL.createObjectURL(blob)
    if (!blob.type.includes('pdf')) {
      tab?.close()
      const a = document.createElement('a')
      a.href = url
      a.download = `shipping-label-${shipmentId}.zpl`
      a.click()
      return null
    }
    if (tab) {
      tab.location.href = url
      return tab
    }
    // The tab failed to open (blocked despite being requested synchronously,
    // or the caller had none to hand in) -- fall back to a plain download
    // so the label isn't simply lost.
    const a = document.createElement('a')
    a.href = url
    a.target = '_blank'
    a.rel = 'noopener'
    a.click()
    return null
  } catch (err) {
    tab?.close()
    throw err
  }
}

// Convenience wrapper for call sites with nothing to await before opening
// the tab (e.g. reprinting an already-created shipment) -- see
// loadShipmentLabelIntoTab's doc for why creation flows must open the
// blank tab themselves, up front, instead of using this.
export function openShipmentLabel(token: string, shipmentId: string, size?: 'A4' | 'A6'): Promise<Window | null> {
  return loadShipmentLabelIntoTab(window.open('', '_blank'), token, shipmentId, size)
}

import { createContext, useContext, type ReactNode } from 'react'

/** The order number an order-confirmation page is about (from its URL). */
const Ctx = createContext<string | null>(null)

export function OrderProvider({ orderNumber, children }: { orderNumber: string; children: ReactNode }) {
  return <Ctx.Provider value={orderNumber}>{children}</Ctx.Provider>
}

export const useOrderNumber = () => useContext(Ctx)

const LAST_ORDER = 'cos-last-order'

/** Checkout remembers (in this tab) who placed which order, so the
 *  confirmation page can show it without asking for the email again. */
export function rememberOrder(orderNumber: string, email: string) {
  try {
    window.sessionStorage.setItem(LAST_ORDER, JSON.stringify({ orderNumber, email }))
  } catch {
    // the confirmation page will ask for the email instead
  }
}

export function rememberedEmailFor(orderNumber: string): string | null {
  try {
    const v = JSON.parse(window.sessionStorage.getItem(LAST_ORDER) ?? 'null') as { orderNumber: string; email: string } | null
    return v?.orderNumber === orderNumber ? v.email : null
  } catch {
    return null
  }
}

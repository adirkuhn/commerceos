// analytics.track(...) is the only thing the rest of the app calls --
// never a vendor SDK directly (no window.gtag/fbq calls anywhere else).
// Swapping the sink (e.g. to a real GA4/Segment adapter) means changing
// this one file.
export type AnalyticsEvent =
  | 'page_view' | 'product_viewed' | 'search' | 'category_viewed'
  | 'add_to_cart' | 'remove_from_cart' | 'checkout_started' | 'checkout_completed' | 'purchase'
  | 'wishlist_added' | 'wishlist_removed' | 'coupon_applied' | 'login' | 'register'

export interface AnalyticsSink {
  track(event: AnalyticsEvent, payload?: Record<string, unknown>): void
}

class ConsoleAnalyticsSink implements AnalyticsSink {
  track(event: AnalyticsEvent, payload?: Record<string, unknown>) {
    if (import.meta.env.MODE === 'test') return
    // eslint-disable-next-line no-console
    console.info(`[analytics] ${event}`, payload ?? {})
  }
}

let sink: AnalyticsSink = new ConsoleAnalyticsSink()

// setAnalyticsSink lets a real deployment swap in a vendor adapter (GA4,
// Segment, PostHog...) at app startup without touching a single call
// site -- see App.tsx for where it would be wired.
export function setAnalyticsSink(next: AnalyticsSink): void {
  sink = next
}

export const analytics: AnalyticsSink = {
  track: (event, payload) => sink.track(event, payload),
}

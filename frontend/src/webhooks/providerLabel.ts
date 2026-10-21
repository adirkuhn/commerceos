// Known providers get a friendlier display name; any future provider this
// codebase doesn't know the name of yet falls back to capitalizing what the
// backend sent -- same "provider is free text" convention as FanOutTab's
// own create form.
const KNOWN_PROVIDER_LABELS: Record<string, string> = { myparcel: 'MyParcel', woocommerce: 'WooCommerce', bol: 'Bol.com', postnl: 'PostNL' }

export function providerLabel(provider: string): string {
  return KNOWN_PROVIDER_LABELS[provider] ?? (provider.charAt(0).toUpperCase() + provider.slice(1))
}

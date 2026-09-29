import type { Connection, SalesChannel, EnrollmentMode, ChannelType, StorefrontLocale, StorefrontMode } from '../types'

export interface ChannelFormState {
  name: string
  channelType: ChannelType
  connectionId: string
  enrollmentMode: EnrollmentMode
  enrollmentCategoryId: string
  enrollmentTag: string
  active: boolean
  defaultWarehouseId: string
  // channelWarehouseId is SalesChannel.default_warehouse_id itself --
  // distinct from defaultWarehouseId above (Connection.default_warehouse_id,
  // which only routes a WooCommerce/Bol import's stock sync). See
  // WAREHOUSE_REQUIRED_TYPES's doc comment for which channel types need it.
  channelWarehouseId: string
  paymentConnectionId: string
  // "" means "use the tenant's default currency" -- only settable at
  // creation, see SalesChannelPayload.currencyCode's own doc comment.
  currencyCode: string
  // Only meaningful (and only sent) for 'internal' channels.
  storefrontMode: StorefrontMode
  // Headless 'internal' channels: the merchant's page that completes
  // shopper sign-in ("" = none). Themed stores use their hosted /login.
  signInUrl: string
  // 'internal' channels: the storefront's default language ("" = the
  // company language, new channels only) and the others shoppers can
  // switch to.
  storefrontLocale: StorefrontLocale | ''
  storefrontLocales: StorefrontLocale[]
}

export function emptyForm(): ChannelFormState {
  return {
    name: '', channelType: 'woocommerce', connectionId: '', enrollmentMode: 'all', enrollmentCategoryId: '',
    enrollmentTag: '', active: true, defaultWarehouseId: '', channelWarehouseId: '', paymentConnectionId: '',
    currencyCode: '', storefrontMode: 'headless', signInUrl: '', storefrontLocale: '', storefrontLocales: [],
  }
}

// defaultWarehouseId lives on the linked Connection, not on the channel
// itself (SalesChannel is purely a product-enrollment rule on top of it --
// see its doc comment) -- so prefilling it means looking the connection up
// by the channel's connection_id.
export function toForm(channel: SalesChannel, connections: Connection[]): ChannelFormState {
  const connection = connections.find(c => c.id === channel.connection_id)
  return {
    name: channel.name,
    channelType: channel.type,
    connectionId: channel.connection_id ?? '',
    enrollmentMode: channel.enrollment_mode,
    enrollmentCategoryId: channel.enrollment_category_id ?? '',
    enrollmentTag: channel.enrollment_tag ?? '',
    active: channel.active,
    defaultWarehouseId: connection?.default_warehouse_id ?? '',
    channelWarehouseId: channel.default_warehouse_id ?? '',
    paymentConnectionId: channel.payment_connection_id ?? '',
    currencyCode: channel.currency_code ?? '',
    storefrontMode: channel.storefront_mode ?? 'headless',
    signInUrl: channel.sign_in_url ?? '',
    storefrontLocale: channel.storefront_locale ?? 'en',
    storefrontLocales: (channel.storefront_locales ?? []).filter(l => l !== (channel.storefront_locale ?? 'en')),
  }
}

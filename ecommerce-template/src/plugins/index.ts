import { pluginRegistry } from './registry'
import { loyaltyPointsPlugin } from './examples/loyaltyPointsPlugin'
import { giftWrapPlugin } from './examples/giftWrapPlugin'

// registerBuiltinPlugins runs once at app startup (see main.tsx). A real
// deployment would call pluginRegistry.register(...) here for whichever
// plugins it ships, first-party or third-party -- nothing else in the
// app needs to change to add or remove one.
export function registerBuiltinPlugins(): void {
  pluginRegistry.register(loyaltyPointsPlugin)
  pluginRegistry.register(giftWrapPlugin)
}

export { pluginRegistry } from './registry'

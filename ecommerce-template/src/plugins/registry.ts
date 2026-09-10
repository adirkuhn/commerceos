import type { EcommercePlugin, HookName, ProductTab, CheckoutExtension, NavigationItem, PageSectionDefinition } from '@/types'

// A deliberately simple plugin registry -- flat arrays merged from every
// registered plugin, read by the places that render them (product page's
// tab list, checkout's step list, layout's nav, page-builder's section
// renderer). No microfrontend machinery: a plugin is just an object
// matching EcommercePlugin, registered once at startup (see App.tsx).
class PluginRegistry {
  private plugins: EcommercePlugin[] = []

  register(plugin: EcommercePlugin): void {
    if (this.plugins.some(p => p.name === plugin.name)) {
      throw new Error(`Plugin "${plugin.name}" is already registered.`)
    }
    this.plugins.push(plugin)
  }

  all(): EcommercePlugin[] {
    return this.plugins
  }

  productTabs(): ProductTab[] {
    return this.plugins.flatMap(p => p.productTabs ?? [])
  }

  checkoutExtensions(step: CheckoutExtension['step']): CheckoutExtension[] {
    return this.plugins.flatMap(p => p.checkoutExtensions ?? []).filter(e => e.step === step)
  }

  navigationItems(position: NavigationItem['position']): NavigationItem[] {
    return this.plugins.flatMap(p => p.navigationItems ?? []).filter(n => n.position === position)
  }

  sections(): PageSectionDefinition[] {
    return this.plugins.flatMap(p => p.sections ?? [])
  }

  dashboardWidgets() {
    return this.plugins.flatMap(p => p.dashboardWidgets ?? [])
  }

  emit(hook: HookName, payload: unknown): void {
    for (const plugin of this.plugins) {
      plugin.hooks?.[hook]?.(payload)
    }
  }
}

export const pluginRegistry = new PluginRegistry()

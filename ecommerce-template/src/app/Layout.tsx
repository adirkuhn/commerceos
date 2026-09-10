import { Outlet } from 'react-router-dom'
import { Header } from '@/components/layout/Header'
import { Footer } from '@/components/layout/Footer'
import { MobileNav } from '@/components/layout/MobileNav'
import { CartDrawer } from '@/components/ecommerce/CartDrawer'
import { Toaster } from '@/components/ui/Toaster'
import { TooltipProvider } from '@/components/ui/Tooltip'

export function Layout() {
  return (
    <TooltipProvider>
      <div className="flex min-h-screen flex-col">
        <a href="#main-content" className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-[100] focus:rounded-md focus:bg-primary focus:px-4 focus:py-2 focus:text-primary-foreground">
          Skip to content
        </a>
        <Header />
        <main id="main-content" className="flex-1">
          <Outlet />
        </main>
        <Footer />
        <CartDrawer />
        <MobileNav />
        <Toaster />
      </div>
    </TooltipProvider>
  )
}

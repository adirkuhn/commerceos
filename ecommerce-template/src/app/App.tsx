import { useEffect } from 'react'
import { BrowserRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ThemeProvider } from '@/themes/ThemeProvider'
import { I18nProvider } from '@/i18n/I18nProvider'
import { AppRouter } from '@/routes/router'
import { registerBuiltinPlugins } from '@/plugins'
import { analytics } from '@/services/analytics'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30 * 1000,
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
})

let pluginsRegistered = false

export function App() {
  useEffect(() => {
    // Registered once, outside React's render cycle in spirit -- guarded
    // here (rather than at module scope) so Vitest's module re-evaluation
    // across test files can't double-register and throw.
    if (!pluginsRegistered) {
      registerBuiltinPlugins()
      pluginsRegistered = true
    }
    analytics.track('page_view', { path: window.location.pathname })
  }, [])

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <I18nProvider>
          <BrowserRouter>
            <AppRouter />
          </BrowserRouter>
        </I18nProvider>
      </ThemeProvider>
    </QueryClientProvider>
  )
}

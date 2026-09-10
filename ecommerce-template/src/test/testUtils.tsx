import type { ReactElement } from 'react'
import { render } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { TooltipProvider } from '@/components/ui/Tooltip'

// A fresh QueryClient per render call -- tests never share cached query
// state with each other, matching how each real page mount gets its own
// query cache lifetime in practice (a single QueryClientProvider at the
// app root, but each test remounts the whole tree).
export function renderWithProviders(ui: ReactElement, { initialEntries = ['/'] }: { initialEntries?: string[] } = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <MemoryRouter initialEntries={initialEntries}>{ui}</MemoryRouter>
      </TooltipProvider>
    </QueryClientProvider>,
  )
}

import { hydrateRoot } from 'react-dom/client'
import { HydrationBoundary, QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter } from 'react-router-dom'
import { StorefrontAPI } from '../api/storefront'
import { ApiProvider } from '../api/ApiContext'
import { App } from './App'
import { STATE_ELEMENT_ID, type PageState } from './state'
import '../theme/base.css'

const stateEl = document.getElementById(STATE_ELEMENT_ID)
if (stateEl?.textContent) {
  const state = JSON.parse(stateEl.textContent) as PageState
  const api = new StorefrontAPI({
    baseURL: state.config.publicApiBaseURL, storeKey: state.config.storeKey, previewToken: state.config.previewToken,
  })
  const qc = new QueryClient({ defaultOptions: { queries: { retry: 1 } } })
  hydrateRoot(
    document.getElementById('root')!,
    <QueryClientProvider client={qc}>
      <HydrationBoundary state={state.dehydrated}>
        <ApiProvider api={api}>
          <BrowserRouter basename="/store">
            <App />
          </BrowserRouter>
        </ApiProvider>
      </HydrationBoundary>
    </QueryClientProvider>,
  )
}

import { Component, type ReactNode } from 'react'
import { Route, Routes, useParams } from 'react-router-dom'
import { StorePage, StoreUnavailable } from './StorePage'

class ErrorBoundary extends Component<{ children: ReactNode }, { failed: boolean }> {
  state = { failed: false }

  static getDerivedStateFromError() {
    return { failed: true }
  }

  componentDidCatch(error: unknown) {
    console.error('storefront: render failed', error)
  }

  render() {
    return this.state.failed ? <StoreUnavailable /> : this.props.children
  }
}

function StoreRoute() {
  const params = useParams()
  return <StorePage path={`/${params['*'] ?? ''}`} storeKey={params.storeKey ?? ''} />
}

/** Routes below the router basename (/store): /{channelKey}/... */
export function App() {
  return (
    <ErrorBoundary>
      <Routes>
        <Route path=":storeKey/*" element={<StoreRoute />} />
        <Route path="*" element={<StoreUnavailable />} />
      </Routes>
    </ErrorBoundary>
  )
}

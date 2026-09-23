import type { ReactElement } from 'react'
import { render } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'

// Renders a routed component the way App.tsx mounts it: under a splat
// route at `mountPath`, starting at `url`. Returns the router so tests can
// assert on the URL and drive history (router.navigate(-1) = device Back).
export function renderAt(ui: ReactElement, mountPath: string, url: string = mountPath) {
  const router = createMemoryRouter([{ path: `${mountPath}/*`, element: ui }], { initialEntries: [url] })
  const result = render(<RouterProvider router={router} />)
  return { ...result, router }
}

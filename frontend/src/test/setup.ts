import '@testing-library/jest-dom/vitest'
import { afterAll, afterEach } from 'vitest'
import { cleanup, configure } from '@testing-library/react'

// findBy*/waitFor default to 1s. Pages now navigate through a data router
// (view state lives in the URL), whose updates commit in a React
// transition -- under a full parallel run that can take longer than 1s on
// a loaded machine, which showed up as intermittent timeouts in tests
// that pass on their own. 3s keeps real failures fast enough to notice.
configure({ asyncUtilTimeout: 3000 })

// Under jsdom, AbortController/AbortSignal are jsdom's, but Request is
// Node's (undici). Newer Node versions reject a foreign signal ("Expected
// signal to be an instance of AbortSignal"), which broke every data-router
// navigation in tests. Tests never abort a router request, so the signal is
// dropped before it reaches undici.
if (typeof window !== 'undefined' && typeof globalThis.Request === 'function') {
  const NodeRequest = globalThis.Request
  globalThis.Request = class extends NodeRequest {
    constructor(input: RequestInfo | URL, init?: RequestInit) {
      if (init?.signal) {
        const { signal: _signal, ...rest } = init
        super(input, rest)
      } else {
        super(input, init)
      }
    }
  } as typeof Request
}

// Unit and component tests must never reach the network: a real request is
// slow, flaky and depends on whatever answers at VITE_API_BASE_URL. Any fetch
// a test did not mock (vi.mock of the api module, or vi.stubGlobal('fetch'))
// lands here: the call rejects, and the test that made it fails afterwards
// even if the app code swallowed the rejection. vi.stubGlobal/unstubAllGlobals
// save and restore this guard, so tests that stub fetch keep working.
const unmockedFetches: string[] = []
globalThis.fetch = ((input: RequestInfo | URL) => {
  const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
  unmockedFetches.push(url)
  return Promise.reject(new Error(`unmocked fetch: ${url}`))
}) as typeof fetch

function failOnUnmockedFetches() {
  if (unmockedFetches.length === 0) return
  const urls = unmockedFetches.splice(0)
  throw new Error(`unmocked fetch: ${urls.join(', ')} -- mock the api module or stub fetch in this test`)
}

afterEach(() => {
  cleanup()
  failOnUnmockedFetches()
})
afterAll(failOnUnmockedFetches)

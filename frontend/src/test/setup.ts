import '@testing-library/jest-dom/vitest'
import { afterEach } from 'vitest'
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

afterEach(() => cleanup())

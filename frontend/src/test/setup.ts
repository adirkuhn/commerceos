import '@testing-library/jest-dom/vitest'
import { afterEach } from 'vitest'
import { cleanup, configure } from '@testing-library/react'

// findBy*/waitFor default to 1s. Pages now navigate through a data router
// (view state lives in the URL), whose updates commit in a React
// transition -- under a full parallel run that can take longer than 1s on
// a loaded machine, which showed up as intermittent timeouts in tests
// that pass on their own. 3s keeps real failures fast enough to notice.
configure({ asyncUtilTimeout: 3000 })

afterEach(() => cleanup())

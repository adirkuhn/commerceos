import i18next from 'i18next'
import { useState } from 'react'
import * as api from '../api'
import { apiErrorText } from '../../api/apiErrorText'

export interface TestResult {
  id: string
  ok: boolean
  message: string
}

// Per-card "Test" button: which connection is being tested and the last
// result (shown on that connection's card). Reloads the list afterwards
// since a test can flip a connection's status.
export function useTestConnection(token: string, onTested: () => void) {
  const [testingId, setTestingId] = useState<string | null>(null)
  const [testResult, setTestResult] = useState<TestResult | null>(null)

  const test = async (id: string) => {
    if (!token) return
    setTestingId(id)
    setTestResult(null)
    try {
      await api.testConnection(token, id)
      setTestResult({ id, ok: true, message: i18next.t('integrations:test.ok') })
    } catch (err) {
      setTestResult({ id, ok: false, message: apiErrorText(err) ?? i18next.t('integrations:test.failed') })
    } finally {
      setTestingId(null)
      onTested()
    }
  }

  return { testingId, testResult, test }
}

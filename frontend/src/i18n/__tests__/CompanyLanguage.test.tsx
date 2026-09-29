// @vitest-environment jsdom
import { render, waitFor } from '@testing-library/react'
import i18next from 'i18next'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { CompanyLanguage } from '../CompanyLanguage'
import type { TenantSettings } from '../../settings/types'

vi.mock('../../settings/api', () => ({ getTenantSettings: vi.fn() }))
import * as api from '../../settings/api'

afterEach(async () => { await i18next.changeLanguage('en') })

describe('CompanyLanguage', () => {
  it('switches the admin to the company language once settings load', async () => {
    vi.mocked(api.getTenantSettings).mockResolvedValue({ language: 'nl' } as TenantSettings)
    render(<CompanyLanguage token="tok" />)

    await waitFor(() => expect(i18next.language).toBe('nl'))
    expect(api.getTenantSettings).toHaveBeenCalledWith('tok')
  })

  it('keeps the current language when settings fail to load', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    vi.mocked(api.getTenantSettings).mockRejectedValue(new Error('offline'))
    render(<CompanyLanguage token="tok" />)

    await waitFor(() => expect(warn).toHaveBeenCalled())
    expect(i18next.language).toBe('en')
  })
})

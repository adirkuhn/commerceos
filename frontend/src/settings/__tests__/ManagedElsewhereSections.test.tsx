// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { describe, it, expect } from 'vitest'
import { AutomationSection, SecuritySection } from '../ManagedElsewhereSections'

describe('SecuritySection', () => {
  it('explains that sign-in is managed in lm2 and links there in a new tab', () => {
    render(<SecuritySection lm2Url="https://lm2.example.com" />)

    expect(screen.getByText(/managed in your lm2 account/)).toBeInTheDocument()
    const link = screen.getByRole('link', { name: /Open lm2/ })
    expect(link).toHaveAttribute('href', 'https://lm2.example.com')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', expect.stringContaining('noopener'))
  })

  it('offers no toggles it cannot enforce', () => {
    render(<SecuritySection lm2Url="https://lm2.example.com" />)
    expect(screen.queryByRole('checkbox')).not.toBeInTheDocument()
  })
})

describe('AutomationSection', () => {
  it('sends people to the real automation rules in Message Center', async () => {
    render(
      <MemoryRouter initialEntries={['/settings/automation']}>
        <Routes>
          <Route path="/settings/automation" element={<AutomationSection />} />
          <Route path="/message-center/rules" element={<p>RULES_PAGE</p>} />
        </Routes>
      </MemoryRouter>,
    )

    await userEvent.click(screen.getByRole('link', { name: /Open automation rules/ }))

    expect(screen.getByText('RULES_PAGE')).toBeInTheDocument()
  })
})

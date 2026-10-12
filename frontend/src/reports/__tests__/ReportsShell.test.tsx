// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import { afterEach, describe, expect, it } from 'vitest'
import { AppProvider } from '../../context/AppContext'
import { ReportsShell } from '../ReportsShell'

function Where() {
  return <p data-testid="where">{useLocation().pathname}</p>
}

function renderShell(url: string) {
  return render(
    <MemoryRouter initialEntries={[url]}>
      <AppProvider>
        <Routes>
          <Route path="*" element={<><ReportsShell><p>report body</p></ReportsShell><Where /></>} />
        </Routes>
      </AppProvider>
    </MemoryRouter>,
  )
}

afterEach(() => document.documentElement.classList.remove('design-depot'))

describe('ReportsShell', () => {
  it('renders the report alone in the classic design', () => {
    renderShell('/reports/customer-rfm')
    expect(screen.getByText('report body')).toBeInTheDocument()
    expect(screen.queryByRole('navigation', { name: 'Reports' })).not.toBeInTheDocument()
  })

  it('lists every report grouped by subject in the depot design, current one marked', () => {
    document.documentElement.classList.add('design-depot')
    renderShell('/reports/customer-rfm')

    const nav = screen.getByRole('navigation', { name: 'Reports' })
    expect(nav).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Customers' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Customer Segments (RFM)' })).toHaveAttribute('aria-current', 'page')
    expect(screen.getByText('report body')).toBeInTheDocument()
  })

  it('opens another report from the list', async () => {
    document.documentElement.classList.add('design-depot')
    renderShell('/reports/customer-rfm')

    await userEvent.click(screen.getByRole('button', { name: 'Stockout Risk' }))

    expect(screen.getByTestId('where').textContent).toBe('/reports/stockout-risk')
  })
})

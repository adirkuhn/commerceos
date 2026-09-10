// @vitest-environment jsdom
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, beforeEach } from 'vitest'
import { Routes, Route } from 'react-router-dom'
import { SearchBar } from '../SearchBar'
import { renderWithProviders } from '@/test/testUtils'
import { useRecentSearchesStore } from '@/stores/recentSearchesStore'

function LocationProbe() {
  return null
}

function renderSearchBar() {
  return renderWithProviders(
    <Routes>
      <Route path="/" element={<SearchBar />} />
      <Route path="/search" element={<LocationProbe />} />
    </Routes>,
  )
}

beforeEach(() => {
  useRecentSearchesStore.setState({ queries: [] })
})

describe('SearchBar', () => {
  it('shows popular searches when focused with an empty query', async () => {
    const user = userEvent.setup()
    renderSearchBar()

    await user.click(screen.getByRole('searchbox', { name: 'Search' }))
    expect(await screen.findByText('Popular searches')).toBeInTheDocument()
    expect(screen.getByText('trail runner')).toBeInTheDocument()
  })

  it('shows product suggestions while typing', async () => {
    const user = userEvent.setup()
    renderSearchBar()

    await user.click(screen.getByRole('searchbox', { name: 'Search' }))
    await user.type(screen.getByRole('searchbox', { name: 'Search' }), 'trail runner')

    expect(await screen.findByText('Suggestions')).toBeInTheDocument()
    expect(await screen.findByText('Traverse Trail Runner')).toBeInTheDocument()
  })

  it('records the query as a recent search on submit', async () => {
    const user = userEvent.setup()
    renderSearchBar()

    await user.type(screen.getByRole('searchbox', { name: 'Search' }), 'headlamp{Enter}')

    await waitFor(() => expect(useRecentSearchesStore.getState().queries).toContain('headlamp'))
  })

  it('shows a clear button once text is entered, which empties the field', async () => {
    const user = userEvent.setup()
    renderSearchBar()

    const input = screen.getByRole('searchbox', { name: 'Search' })
    await user.type(input, 'boots')
    await user.click(screen.getByRole('button', { name: 'Clear search' }))

    expect(input).toHaveValue('')
  })
})

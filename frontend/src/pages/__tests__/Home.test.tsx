// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

vi.mock('../../dashboard/TodayPage', () => ({ default: () => <p>today page</p> }))

import Home from '../Home'

describe('Home', () => {
  it('opens on Today', () => {
    render(<Home />)
    expect(screen.getByText('today page')).toBeInTheDocument()
  })
})

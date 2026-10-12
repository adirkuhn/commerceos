// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

vi.mock('../Dashboard', () => ({ default: () => <p>classic dashboard</p> }))
vi.mock('../../dashboard/TodayPage', () => ({ default: () => <p>today page</p> }))

import Home from '../Home'

afterEach(() => document.documentElement.classList.remove('design-depot'))

describe('Home', () => {
  it('opens on Today in the depot design', () => {
    document.documentElement.classList.add('design-depot')
    render(<Home />)
    expect(screen.getByText('today page')).toBeInTheDocument()
  })

  it('keeps the classic dashboard in the classic design', () => {
    render(<Home />)
    expect(screen.getByText('classic dashboard')).toBeInTheDocument()
  })
})

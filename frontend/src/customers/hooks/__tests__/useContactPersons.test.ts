// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'

vi.mock('../../api', () => ({
  listContactPersons: vi.fn(), addContactPerson: vi.fn(), updateContactPerson: vi.fn(), removeContactPerson: vi.fn(),
}))
import * as api from '../../api'
import { useContactPersons } from '../useContactPersons'

const kylian = { id: 'cp-1', customer_id: 'c-1', first_name: 'Kylian', last_name: '', role: '', email: '', phone: '', created_at: '' }

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.listContactPersons).mockResolvedValue([kylian])
})

describe('useContactPersons', () => {
  it('loads a customer\'s contact persons', async () => {
    const { result } = renderHook(() => useContactPersons('tok', 'c-1'))
    await waitFor(() => expect(result.current.persons).toEqual([kylian]))
    expect(api.listContactPersons).toHaveBeenCalledWith('tok', 'c-1')
  })

  it('adds, updates and removes, keeping the list in step', async () => {
    const { result } = renderHook(() => useContactPersons('tok', 'c-1'))
    await waitFor(() => expect(result.current.persons).toHaveLength(1))

    vi.mocked(api.addContactPerson).mockResolvedValue({ ...kylian, id: 'cp-2', first_name: 'Sara' })
    await act(() => result.current.add({ first_name: 'Sara' }))
    expect(result.current.persons.map(p => p.first_name)).toEqual(['Kylian', 'Sara'])

    vi.mocked(api.updateContactPerson).mockResolvedValue({ ...kylian, phone: '0612' })
    await act(() => result.current.update('cp-1', { first_name: 'Kylian', phone: '0612' }))
    expect(result.current.persons[0].phone).toBe('0612')

    vi.mocked(api.removeContactPerson).mockResolvedValue(null)
    await act(() => result.current.remove('cp-2'))
    expect(api.removeContactPerson).toHaveBeenCalledWith('tok', 'c-1', 'cp-2')
    expect(result.current.persons.map(p => p.id)).toEqual(['cp-1'])
  })

  it('shows a load error', async () => {
    vi.mocked(api.listContactPersons).mockRejectedValue(new Error('boom'))
    const { result } = renderHook(() => useContactPersons('tok', 'c-1'))
    await waitFor(() => expect(result.current.error).toBe('boom'))
  })

  it('loads nothing without a customer', () => {
    renderHook(() => useContactPersons('tok', null))
    expect(api.listContactPersons).not.toHaveBeenCalled()
  })
})

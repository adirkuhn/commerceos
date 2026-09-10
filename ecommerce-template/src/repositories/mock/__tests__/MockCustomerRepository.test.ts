import { describe, expect, it } from 'vitest'
import { MockCustomerRepository } from '../MockCustomerRepository'
import { DEMO_CUSTOMER, DEMO_PASSWORD } from '@/config/demoData'

const repo = new MockCustomerRepository()

describe('MockCustomerRepository', () => {
  it('logs in the seeded demo account', async () => {
    const customer = await repo.login(DEMO_CUSTOMER.email, DEMO_PASSWORD)
    expect(customer.id).toBe(DEMO_CUSTOMER.id)
  })

  it('is case-insensitive on email', async () => {
    const customer = await repo.login(DEMO_CUSTOMER.email.toUpperCase(), DEMO_PASSWORD)
    expect(customer.id).toBe(DEMO_CUSTOMER.id)
  })

  it('rejects a wrong password', async () => {
    await expect(repo.login(DEMO_CUSTOMER.email, 'wrong-password')).rejects.toThrow()
  })

  it('rejects an unknown email', async () => {
    await expect(repo.login('nobody@example.com', 'anything')).rejects.toThrow()
  })

  it('registers a new account and can log in with it', async () => {
    const email = `test-${Date.now()}@example.com`
    const created = await repo.register({ firstName: 'Test', lastName: 'User', email, password: 'hunter22' })
    expect(created.email).toBe(email)
    const loggedIn = await repo.login(email, 'hunter22')
    expect(loggedIn.id).toBe(created.id)
  })

  it('rejects registering an email that already exists', async () => {
    await expect(repo.register({ firstName: 'Dup', lastName: 'User', email: DEMO_CUSTOMER.email, password: 'whatever1' })).rejects.toThrow()
  })

  it('adds, updates, and removes an address', async () => {
    const email = `addr-${Date.now()}@example.com`
    const customer = await repo.register({ firstName: 'Addr', lastName: 'Test', email, password: 'password1' })
    const withAddress = await repo.addAddress(customer.id, {
      firstName: 'Addr', lastName: 'Test', line1: '1 Main St', city: 'Boulder', postalCode: '80302', country: 'US',
    })
    expect(withAddress.addresses).toHaveLength(1)
    const addressId = withAddress.addresses[0].id

    const updated = await repo.updateAddress(customer.id, { ...withAddress.addresses[0], city: 'Denver' })
    expect(updated.addresses[0].city).toBe('Denver')

    const removed = await repo.removeAddress(customer.id, addressId)
    expect(removed.addresses).toHaveLength(0)
  })
})

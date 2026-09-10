import type { Customer, Address } from '@/types'
import type { CustomerRepository } from '../types'
import { simulateLatency } from './latency'
import { DEMO_CUSTOMER, DEMO_PASSWORD } from '@/config/demoData'

interface Account {
  customer: Customer
  password: string
}

const accounts = new Map<string, Account>([[DEMO_CUSTOMER.email, { customer: DEMO_CUSTOMER, password: DEMO_PASSWORD }]])
const byId = new Map<string, string>([[DEMO_CUSTOMER.id, DEMO_CUSTOMER.email]])

let customerSeq = 0
let addressSeq = 0

export class MockCustomerRepository implements CustomerRepository {
  async login(email: string, password: string): Promise<Customer> {
    await simulateLatency()
    const account = accounts.get(email.toLowerCase().trim())
    if (!account || account.password !== password) {
      throw new Error('Incorrect email or password.')
    }
    return account.customer
  }

  async register(input: { firstName: string; lastName: string; email: string; password: string }): Promise<Customer> {
    await simulateLatency()
    const email = input.email.toLowerCase().trim()
    if (accounts.has(email)) throw new Error('An account with this email already exists.')
    customerSeq += 1
    const customer: Customer = {
      id: `cust-${customerSeq}`, firstName: input.firstName, lastName: input.lastName, email,
      addresses: [], preferences: { orderUpdates: true, newsletter: false, productRestock: false },
      createdAt: new Date().toISOString(),
    }
    accounts.set(email, { customer, password: input.password })
    byId.set(customer.id, email)
    return customer
  }

  async getById(id: string): Promise<Customer | null> {
    await simulateLatency()
    const email = byId.get(id)
    return email ? accounts.get(email)?.customer ?? null : null
  }

  async update(id: string, patch: Partial<Customer>): Promise<Customer> {
    await simulateLatency()
    const account = this.accountById(id)
    account.customer = { ...account.customer, ...patch }
    return account.customer
  }

  async addAddress(id: string, address: Omit<Address, 'id'>): Promise<Customer> {
    await simulateLatency()
    const account = this.accountById(id)
    addressSeq += 1
    account.customer = { ...account.customer, addresses: [...account.customer.addresses, { ...address, id: `addr-new-${addressSeq}` }] }
    return account.customer
  }

  async updateAddress(id: string, address: Address): Promise<Customer> {
    await simulateLatency()
    const account = this.accountById(id)
    account.customer = {
      ...account.customer,
      addresses: account.customer.addresses.map(a => (a.id === address.id ? address : a)),
    }
    return account.customer
  }

  async removeAddress(id: string, addressId: string): Promise<Customer> {
    await simulateLatency()
    const account = this.accountById(id)
    account.customer = { ...account.customer, addresses: account.customer.addresses.filter(a => a.id !== addressId) }
    return account.customer
  }

  private accountById(id: string): Account {
    const email = byId.get(id)
    const account = email ? accounts.get(email) : undefined
    if (!account) throw new Error('Customer not found.')
    return account
  }
}

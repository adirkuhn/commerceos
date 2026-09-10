import { customerRepository } from '@/repositories'
import type { Customer, Address } from '@/types'

export const loginCustomer = (email: string, password: string) => customerRepository.login(email, password)
export const registerCustomer = (input: { firstName: string; lastName: string; email: string; password: string }) =>
  customerRepository.register(input)
export const getCustomerById = (id: string) => customerRepository.getById(id)
export const updateCustomer = (id: string, patch: Partial<Customer>) => customerRepository.update(id, patch)
export const addCustomerAddress = (id: string, address: Omit<Address, 'id'>) => customerRepository.addAddress(id, address)
export const updateCustomerAddress = (id: string, address: Address) => customerRepository.updateAddress(id, address)
export const removeCustomerAddress = (id: string, addressId: string) => customerRepository.removeAddress(id, addressId)

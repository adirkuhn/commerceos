import { useMutation, useQueryClient } from '@tanstack/react-query'
import { updateCustomer, addCustomerAddress, updateCustomerAddress, removeCustomerAddress } from '@/api/customers'
import type { Customer, Address } from '@/types'
import { useAuthStore } from '@/stores/authStore'

export function useUpdateProfile() {
  const setCustomer = useAuthStore(s => s.setCustomer)
  const customerId = useAuthStore(s => s.session?.customer.id)
  return useMutation({
    mutationFn: (patch: Partial<Customer>) => updateCustomer(customerId!, patch),
    onSuccess: setCustomer,
  })
}

export function useAddAddress() {
  const setCustomer = useAuthStore(s => s.setCustomer)
  const customerId = useAuthStore(s => s.session?.customer.id)
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (address: Omit<Address, 'id'>) => addCustomerAddress(customerId!, address),
    onSuccess: (customer) => { setCustomer(customer); queryClient.invalidateQueries({ queryKey: ['customer'] }) },
  })
}

export function useUpdateAddress() {
  const setCustomer = useAuthStore(s => s.setCustomer)
  const customerId = useAuthStore(s => s.session?.customer.id)
  return useMutation({
    mutationFn: (address: Address) => updateCustomerAddress(customerId!, address),
    onSuccess: setCustomer,
  })
}

export function useRemoveAddress() {
  const setCustomer = useAuthStore(s => s.setCustomer)
  const customerId = useAuthStore(s => s.session?.customer.id)
  return useMutation({
    mutationFn: (addressId: string) => removeCustomerAddress(customerId!, addressId),
    onSuccess: setCustomer,
  })
}

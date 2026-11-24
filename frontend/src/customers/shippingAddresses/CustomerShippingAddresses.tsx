import { useAuth } from '../../auth/AuthContext'
import { useShippingAddresses } from '../hooks/useShippingAddresses'
import { ShippingAddressesCard } from './ShippingAddressesCard'

// A saved customer's shipping address book, loaded and saved on its own.
export function CustomerShippingAddresses({ customerId }: { customerId: string }) {
  const { token } = useAuth()
  const book = useShippingAddresses(token ?? '', customerId)
  return (
    <ShippingAddressesCard addresses={book.addresses} error={book.error}
      onAdd={book.add} onUpdate={book.update} onRemove={book.remove} onMakeDefault={book.makeDefault} />
  )
}

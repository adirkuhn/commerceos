import { Link } from 'react-router-dom'
import { useCart } from '../../cart/CartContext'
import { useWishlist } from '../../wishlist/WishlistContext'
import { useAuth } from '../../auth/AuthContext'

// NavActionsBlock is the header page's one required functional block (see
// backend domain/theme.BlockTypeNavActions' own doc comment) -- the real,
// live Cart/Wishlist/Account links, bundled as one wired widget rather
// than split into three blocks, since none of them makes sense placed
// independently of the others. Relocated from App.tsx's own former
// Layout (CartLink/WishlistLink/AccountLink) -- same behavior/styling,
// just reachable from BlockRenderer/puck config now.
export function NavActionsBlock() {
  const { cart } = useCart()
  const { wishlist } = useWishlist()
  const { account, loading } = useAuth()
  const cartCount = cart?.items.reduce((sum, i) => sum + i.quantity, 0) ?? 0
  const wishlistCount = wishlist?.product_ids.length ?? 0

  return (
    <div className="flex h-full items-center gap-6 text-sm">
      <Link to="/cart" className="text-brand-500 hover:text-brand-700">
        Cart{cartCount > 0 ? ` (${cartCount})` : ''}
      </Link>
      <Link to="/wishlist" className="text-brand-500 hover:text-brand-700">
        Wishlist{wishlistCount > 0 ? ` (${wishlistCount})` : ''}
      </Link>
      {!loading && (
        <Link to={account ? '/account' : '/login'} className="text-brand-500 hover:text-brand-700">
          {account ? (account.first_name || 'Account') : 'Sign in'}
        </Link>
      )}
    </div>
  )
}

// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Suspense, type ReactNode } from 'react'
import { rendererFor } from '../renderers'
import { RenderProvider, type RenderEnv } from '../../runtime/RenderContext'
import { ApiProvider } from '../../api/ApiContext'
import { StorefrontAPI, type Cart, type ProductDetail } from '../../api/storefront'
import { CartProvider } from '../../runtime/page/CartContext'
import { CustomerProvider } from '../../runtime/page/CustomerContext'
import { WishlistProvider } from '../../runtime/page/WishlistContext'
import { OrderProvider, rememberOrder } from '../../runtime/page/OrderContext'
import { ProductProvider } from '../../runtime/page/ProductContext'
import { StoreInfoProvider } from '../../runtime/page/StoreContext'
import { defaultSettings } from '../../core/settings'

const cartWithItem: Cart = {
  token: 'cart-1', subtotal_minor_units: 5000, discount_minor_units: 0,
  items: [{ sku_code: 'RS', product_name: 'Red shoes', quantity: 1, price_minor_units: 5000, currency_code: 'EUR' }],
}

const product: ProductDetail = { id: 'p1', name: 'Red shoes', slug: 'red-shoes', description: '', category_id: null, tags: [], images: [], variants: [] }

function fakeAPI() {
  const api = new StorefrontAPI({ baseURL: 'x', storeKey: 'k' })
  api.getCart = vi.fn(async () => cartWithItem)
  api.shippingOptions = vi.fn(async () => [{ title: 'PostNL', amount_minor_units: 495, currency_code: 'EUR' }])
  api.checkout = vi.fn(async () => ({ order_number: 'CO-7', total_minor_units: 5495, currency_code: 'EUR', payment_url: 'https://pay.test/s/1' }))
  api.lookupOrder = vi.fn(async () => ({ order_number: 'CO-7', status: 'paid', fulfilled: false, total_minor_units: 5495, currency_code: 'EUR' }))
  api.subscribeNewsletter = vi.fn(async () => ({ status: 'check_your_email' as const }))
  api.requestMagicLink = vi.fn(async () => undefined)
  api.verifySession = vi.fn(async () => ({ session_token: 'sess', account: { email: 'ann@x.test', first_name: 'Ann', last_name: 'B' } }))
  api.createWishlist = vi.fn(async () => ({ token: 'wl', product_ids: [] }))
  api.toggleWishlistItem = vi.fn(async () => ({ token: 'wl', product_ids: ['p1'] }))
  return api
}

function setup(el: ReactNode, opts: { api?: StorefrontAPI; env?: Partial<RenderEnv>; order?: string; product?: boolean } = {}) {
  const api = opts.api ?? fakeAPI()
  const navigate = vi.fn()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  let tree = el
  if (opts.order) tree = <OrderProvider orderNumber={opts.order}>{tree}</OrderProvider>
  if (opts.product) tree = <ProductProvider product={product}>{tree}</ProductProvider>
  render(
    <QueryClientProvider client={qc}>
      <ApiProvider api={api}>
        <RenderProvider value={{ storePath: p => `/store/k/${p}`, navigate, ...opts.env }}>
          <StoreInfoProvider value={{ name: 'Acme Goods', settings: defaultSettings(), assets: {} }}>
            <CustomerProvider api={api} storeKey="k">
              <WishlistProvider api={api} storeKey="k">
                <CartProvider api={api} storeKey="k"><Suspense fallback={null}>{tree}</Suspense></CartProvider>
              </WishlistProvider>
            </CustomerProvider>
          </StoreInfoProvider>
        </RenderProvider>
      </ApiProvider>
    </QueryClientProvider>,
  )
  return { api, navigate }
}

const R = (t: string) => rendererFor(t)!
const node = (type: string) => ({ id: type, type })

beforeEach(() => {
  window.localStorage.clear()
  window.sessionStorage.clear()
  window.history.replaceState({}, '', '/store/k/')
})

async function fillAddress() {
  await userEvent.type(await screen.findByLabelText('Email'), 'ann@x.test')
  await userEvent.type(screen.getByLabelText('First name'), 'Ann')
  await userEvent.type(screen.getByLabelText('Last name'), 'B')
  await userEvent.type(screen.getByLabelText('Street and number'), 'Main 1')
  await userEvent.type(screen.getByLabelText('Postcode'), '3511 AA')
  await userEvent.type(screen.getByLabelText('City'), 'Utrecht')
}

describe('checkout form', () => {
  beforeEach(() => window.localStorage.setItem('cos-cart:k', 'cart-1'))

  it('prices delivery, places the order and sends the shopper to the payment page', async () => {
    const Form = R('checkout-form')
    const { api, navigate } = setup(<Form node={node('checkout-form')} props={{ button_label: 'Pay now' }} className="" />)
    await fillAddress()
    expect(await screen.findByRole('radio', { name: /PostNL — €4.95/ })).toBeChecked()
    expect(screen.getByText('€54.95')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Pay now' }))
    await waitFor(() => expect(navigate).toHaveBeenCalledWith('https://pay.test/s/1'))
    expect(api.checkout).toHaveBeenCalledWith('cart-1', expect.objectContaining({ email: 'ann@x.test', shipping_method_title: 'PostNL' }))
    expect(window.sessionStorage.getItem('cos-last-order')).toContain('CO-7')
    expect(window.localStorage.getItem('cos-cart:k')).toBeNull()
  })

  it('never follows a non-https payment link: goes to the order page instead', async () => {
    const api = fakeAPI()
    api.checkout = vi.fn(async () => ({ order_number: 'CO-8', total_minor_units: 1, currency_code: 'EUR', payment_url: 'javascript:alert(1)' }))
    const Form = R('checkout-form')
    const { navigate } = setup(<Form node={node('checkout-form')} props={{ button_label: 'Pay' }} className="" />, { api })
    await fillAddress()
    await userEvent.click(screen.getByRole('button', { name: 'Pay' }))
    await waitFor(() => expect(navigate).toHaveBeenCalledWith('/store/k/orders/CO-8'))
  })

  it('shows why an order could not be placed', async () => {
    const api = fakeAPI()
    api.checkout = vi.fn(async () => { throw new Error('Out of stock') })
    const Form = R('checkout-form')
    setup(<Form node={node('checkout-form')} props={{ button_label: 'Pay' }} className="" />, { api })
    await fillAddress()
    await userEvent.click(screen.getByRole('button', { name: 'Pay' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Out of stock')
  })
})

describe('order status', () => {
  it('shows the order right away to the shopper who placed it', async () => {
    rememberOrder('CO-7', 'ann@x.test')
    const Status = R('order-status')
    const { api } = setup(<Status node={node('order-status')} props={{ title: 'Thanks' }} className="" />, { order: 'CO-7' })
    expect(await screen.findByText('paid')).toBeInTheDocument()
    expect(api.lookupOrder).toHaveBeenCalledWith('CO-7', 'ann@x.test')
  })

  it('asks anyone else for the order email', async () => {
    const Status = R('order-status')
    const { api } = setup(<Status node={node('order-status')} props={{ title: 'Thanks' }} className="" />, { order: 'CO-7' })
    expect(api.lookupOrder).not.toHaveBeenCalled()
    await userEvent.type(screen.getByLabelText(/email you ordered with/), 'ann@x.test')
    await userEvent.click(screen.getByRole('button', { name: 'Show order' }))
    expect(await screen.findByText('€54.95')).toBeInTheDocument()
  })
})

describe('newsletter signup', () => {
  it('needs consent, then asks the shopper to confirm by email', async () => {
    const Signup = R('newsletter-signup')
    const { api } = setup(<Signup node={node('newsletter-signup')} props={{ title: 'News', text: '', button_label: 'Join', consent_label: 'Yes, email me' }} className="" />)
    await userEvent.type(screen.getByLabelText('Email'), 'ann@x.test')
    expect(screen.getByRole('button', { name: 'Join' })).toBeDisabled()
    await userEvent.click(screen.getByLabelText('Yes, email me'))
    await userEvent.click(screen.getByRole('button', { name: 'Join' }))
    expect(await screen.findByRole('status')).toHaveTextContent('confirm your subscription')
    expect(api.subscribeNewsletter).toHaveBeenCalledWith('ann@x.test', true)
  })
})

describe('sign in', () => {
  it('asks for a sign-in link without choosing where it points', async () => {
    const Login = R('login-form')
    const { api } = setup(<Login node={node('login-form')} props={{ title: 'Sign in', button_label: 'Send link' }} className="" />)
    await userEvent.type(screen.getByLabelText('Email'), 'ann@x.test')
    await userEvent.click(screen.getByRole('button', { name: 'Send link' }))
    expect(await screen.findByRole('status')).toHaveTextContent('Check your inbox')
    expect(api.requestMagicLink).toHaveBeenCalledWith('ann@x.test')
  })

  it('completes sign-in from the emailed link and goes to the account', async () => {
    window.history.replaceState({}, '', '/store/k/login?token=link-1')
    const Login = R('login-form')
    const { api, navigate } = setup(<Login node={node('login-form')} props={{ title: 'Sign in', button_label: 'Send' }} className="" />)
    await waitFor(() => expect(navigate).toHaveBeenCalledWith('/store/k/account', { replace: true }))
    expect(api.verifySession).toHaveBeenCalledWith('link-1')
    expect(window.localStorage.getItem('cos-session:k')).toBe('sess')
  })
})

describe('site components', () => {
  it('logo falls back to the store name; menu keeps only safe links', () => {
    const [Logo, Menu] = [R('site-logo'), R('navigation-menu')]
    setup(<><Logo node={node('site-logo')} props={{ size: 'md' }} className="" />
      <Menu node={node('navigation-menu')} props={{ orientation: 'horizontal', items: [{ label: 'Sale', link: '/collections/sale' }, { label: 'Bad', link: 'javascript:alert(1)' }] }} className="" /></>)
    expect(screen.getByRole('link', { name: 'Acme Goods' })).toHaveAttribute('href', '/store/k/')
    expect(screen.getByRole('link', { name: 'Sale' })).toHaveAttribute('href', '/store/k/collections/sale')
    expect(screen.queryByRole('link', { name: 'Bad' })).toBeNull()
  })

  it('wishlist button saves the product', async () => {
    const Button = R('wishlist-button')
    const { api } = setup(<Button node={node('wishlist-button')} props={{ add_label: 'Save', remove_label: 'Saved' }} className="" />, { product: true })
    await userEvent.click(await screen.findByRole('button', { name: 'Save' }))
    expect(await screen.findByRole('button', { name: 'Saved' })).toHaveAttribute('aria-pressed', 'true')
    expect(api.toggleWishlistItem).toHaveBeenCalledWith('wl', 'p1')
  })
})

describe('product tile', () => {
  const base = { id: 'p1', name: 'Feijão Preto', slug: 'feijao', category_id: null, tags: [], image_url: null, currency_code: 'EUR' }

  it('shows the discount, stock and adds a single-SKU product to the cart', async () => {
    const { ProductTile, percentOff } = await import('../commerce/ProductTile')
    expect(percentOff(500, 310)).toBe(38)
    expect(percentOff(500, 500)).toBeUndefined()
    const api = fakeAPI()
    api.createCart = vi.fn(async () => ({ token: 'c', items: [], subtotal_minor_units: 0, discount_minor_units: 0 }))
    api.setCartItems = vi.fn(async (_t: string, items: { sku_code: string; quantity: number }[]) => ({ ...cartWithItem, items: [] , token: 'c', subtotal_minor_units: items.length }))
    const { api: used } = setup(<ProductTile product={{ ...base, price_minor_units: 500, sale_price_minor_units: 310, sku_code: 'FEIJAO-1KG', sku_count: 1, in_stock: true }} showPrice showDiscount showStock showAddToCart />, { api })
    expect(screen.getByText('-38%')).toBeInTheDocument()
    expect(screen.getByText('In stock')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Add to cart' }))
    await waitFor(() => expect(used.setCartItems).toHaveBeenCalledWith('c', [{ sku_code: 'FEIJAO-1KG', quantity: 1 }]))
  })

  it('sends products with options to their page and disables sold-out ones', async () => {
    const { ProductTile } = await import('../commerce/ProductTile')
    setup(<>
      <ProductTile product={{ ...base, id: 'a', slug: 'tee', name: 'Tee', price_minor_units: 900, sku_code: null, sku_count: 3, in_stock: true }} showPrice showAddToCart />
      <ProductTile product={{ ...base, id: 'b', slug: 'gone', name: 'Gone', price_minor_units: 900, sku_code: 'G', sku_count: 1, in_stock: false }} showPrice showAddToCart />
    </>)
    expect(screen.getByRole('link', { name: 'Choose options' })).toHaveAttribute('href', '/store/k/products/tee')
    expect(screen.getByRole('button', { name: 'Sold out' })).toBeDisabled()
  })
})

describe('social links', () => {
  it('links to each network with an accessible name, dropping unsafe links', () => {
    const Social = R('social-links')
    setup(<Social node={node('social-links')} props={{ size: 'md', show_labels: false, items: [
      { network: 'whatsapp', link: 'https://wa.me/31640140482' },
      { network: 'maps', link: 'https://maps.google.com/?q=Purmerplein+28A' },
      { network: 'instagram', link: 'javascript:alert(1)' },
    ] }} className="" />)
    expect(screen.getByRole('link', { name: 'WhatsApp' })).toHaveAttribute('href', 'https://wa.me/31640140482')
    expect(screen.getByRole('link', { name: 'WhatsApp' })).toHaveAttribute('rel', 'noopener noreferrer')
    expect(screen.getByRole('link', { name: 'Map' })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Instagram' })).toBeNull()
  })
})

// Realistic demo catalog for "Aurora" -- an outdoor/lifestyle goods store.
// Every mock repository (src/repositories) reads from this module. This
// is the one place fixture data lives; nothing here is Lorem Ipsum.
import type {
  Brand, Category, Product, ProductVariant, Order, Customer, Coupon, Review, Question, FlashSale, BlogPost, CmsPage,
  Company, CompanyUserSummary, PriceList,
} from '@/types'
import { slugify } from '@/lib/utils'

export const CATEGORIES: Category[] = [
  { id: 'cat-outerwear', name: 'Outerwear', slug: 'outerwear', description: 'Shells, insulated jackets, and rain gear built for real weather.', imageUrl: 'https://images.unsplash.com/photo-1551028719-00167b16eac5?w=800' },
  { id: 'cat-footwear', name: 'Footwear', slug: 'footwear', description: 'Trail runners, hiking boots, and everyday sneakers.', imageUrl: 'https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=800' },
  { id: 'cat-bags', name: 'Bags & Packs', slug: 'bags-packs', description: 'Daypacks, duffels, and travel packs.', imageUrl: 'https://images.unsplash.com/photo-1553062407-98eeb64c6a62?w=800' },
  { id: 'cat-apparel', name: 'Apparel', slug: 'apparel', description: 'Base layers, fleece, and everyday tees.', imageUrl: 'https://images.unsplash.com/photo-1523381210434-271e8be1f52b?w=800' },
  { id: 'cat-accessories', name: 'Accessories', slug: 'accessories', description: 'Hats, gloves, belts, and everyday carry.', imageUrl: 'https://images.unsplash.com/photo-1553062407-98eeb64c6a62?w=800' },
  { id: 'cat-home', name: 'Home & Living', slug: 'home-living', description: 'Drinkware, blankets, and camp kitchen.', imageUrl: 'https://images.unsplash.com/photo-1493663284031-b7e3aefcae8e?w=800' },
  { id: 'cat-electronics', name: 'Electronics', slug: 'electronics', description: 'Headlamps, trackers, and portable power.', imageUrl: 'https://images.unsplash.com/photo-1518770660439-4636190af475?w=800' },
  { id: 'cat-fitness', name: 'Fitness', slug: 'fitness', description: 'Training gear for the trail and the gym.', imageUrl: 'https://images.unsplash.com/photo-1517836357463-d25dfeac3438?w=800' },
]

export const BRANDS: Brand[] = [
  { id: 'brand-northfield', name: 'Northfield', slug: 'northfield', description: 'Technical outerwear made for alpine conditions.' },
  { id: 'brand-solstice', name: 'Solstice', slug: 'solstice', description: 'Lightweight performance apparel for warm-weather trails.' },
  { id: 'brand-rivet', name: 'Rivet & Co.', slug: 'rivet-co', description: 'Heritage-inspired bags and leather goods.' },
  { id: 'brand-aerowave', name: 'Aerowave', slug: 'aerowave', description: 'Consumer electronics for outdoor life.' },
  { id: 'brand-cairn', name: 'Cairn', slug: 'cairn', description: 'Minimal home goods for camp and kitchen.' },
  { id: 'brand-fieldstone', name: 'Fieldstone', slug: 'fieldstone', description: 'Footwear engineered for long days on trail.' },
]

let productSeq = 0
function nextId(prefix: string): string {
  productSeq += 1
  return `${prefix}-${productSeq}`
}

const SIZE_OPTIONS = ['S', 'M', 'L', 'XL']
const COLOR_OPTIONS: Record<string, string[]> = {
  jacket: ['Charcoal', 'Moss', 'Rust'],
  shoe: ['Black', 'Stone', 'Ember'],
  tee: ['Black', 'Navy', 'Sand'],
}

function image(url: string, alt: string) {
  return { id: nextId('img'), url, alt }
}

interface BuildProductInput {
  name: string
  category: string
  brand?: string
  price: number
  compareAtPrice?: number
  shortDescription: string
  description: string
  imageUrl: string
  variantKind?: 'jacket' | 'shoe' | 'tee' | null
  stock?: number
  tags?: string[]
  isFeatured?: boolean
  isNewArrival?: boolean
  isBestSeller?: boolean
  kind?: Product['kind']
  attributes?: { name: string; value: string }[]
  weightGrams?: number
}

function buildProduct(input: BuildProductInput): Product {
  const id = nextId('prod')
  const slug = slugify(input.name)
  const sku = `${input.category.slice(0, 3).toUpperCase()}-${id.split('-')[1]}`
  const variants: ProductVariant[] = []
  const variantOptionNames: string[] = []

  if (input.variantKind) {
    variantOptionNames.push('Size', 'Color')
    const colors = COLOR_OPTIONS[input.variantKind]
    let vi = 0
    for (const size of SIZE_OPTIONS) {
      for (const color of colors) {
        vi += 1
        const stock = (vi * 7) % 15
        variants.push({
          id: `${id}-v${vi}`,
          sku: `${sku}-${size}-${color.slice(0, 3).toUpperCase()}`,
          options: [{ name: 'Size', value: size }, { name: 'Color', value: color }],
          price: input.price,
          compareAtPrice: input.compareAtPrice,
          stock,
        })
      }
    }
  }

  const totalStock = variants.length > 0 ? variants.reduce((sum, v) => sum + v.stock, 0) : (input.stock ?? 24)
  const stockStatus: Product['stockStatus'] = totalStock === 0 ? 'out_of_stock' : totalStock <= 5 ? 'low_stock' : 'in_stock'

  return {
    id, slug, name: input.name, sku, kind: input.kind ?? (variants.length > 0 ? 'variable' : 'simple'),
    brandId: input.brand, categoryIds: [input.category], tags: input.tags ?? [],
    shortDescription: input.shortDescription, description: input.description,
    images: [image(input.imageUrl, input.name)],
    price: input.price, compareAtPrice: input.compareAtPrice, currency: 'USD', taxIncluded: false,
    attributes: input.attributes ?? [],
    variantOptionNames, variants,
    stock: totalStock, stockStatus, lowStockThreshold: 5,
    rating: 0, reviewCount: 0,
    isFeatured: input.isFeatured ?? false, isNewArrival: input.isNewArrival ?? false, isBestSeller: input.isBestSeller ?? false,
    weightGrams: input.weightGrams,
    createdAt: new Date(2026, 0, 1 + (productSeq % 200)).toISOString(),
  }
}

export const PRODUCTS: Product[] = [
  buildProduct({ name: 'Summit Shell Jacket', category: 'cat-outerwear', brand: 'brand-northfield', price: 24900, compareAtPrice: 29900, shortDescription: '3-layer waterproof shell for alpine weather.', description: 'A fully seam-sealed 3-layer shell built for sustained exposure to wind and rain. Pit zips, helmet-compatible hood, and a drop-tail hem.', imageUrl: 'https://images.unsplash.com/photo-1551028719-00167b16eac5?w=800', variantKind: 'jacket', isFeatured: true, isBestSeller: true, tags: ['waterproof', 'shell'], weightGrams: 420 }),
  buildProduct({ name: 'Ridgeline Down Parka', category: 'cat-outerwear', brand: 'brand-northfield', price: 32900, shortDescription: '700-fill down parka for deep cold.', description: 'Traps warmth with responsibly-sourced 700-fill down and a windproof shell face. Rated to -20°C for stationary use.', imageUrl: 'https://images.unsplash.com/photo-1544923246-77307dd654cb?w=800', variantKind: 'jacket', isNewArrival: true, tags: ['down', 'winter'] }),
  buildProduct({ name: 'Windward Packable Anorak', category: 'cat-outerwear', brand: 'brand-solstice', price: 8900, shortDescription: 'Packs into its own pocket, weighs almost nothing.', description: 'A wind-resistant anorak that compresses into its chest pocket. The layer you always have with you and never notice until you need it.', imageUrl: 'https://images.unsplash.com/photo-1591047139829-d91aecb6caea?w=800', variantKind: 'jacket', tags: ['packable', 'wind'] }),
  buildProduct({ name: 'Basecamp Fleece Half-Zip', category: 'cat-outerwear', brand: 'brand-solstice', price: 6900, compareAtPrice: 8500, shortDescription: 'Midweight grid fleece for camp and commute.', description: 'A grid-fleece half-zip that regulates temperature whether you are hiking uphill or sitting around a fire.', imageUrl: 'https://images.unsplash.com/photo-1516826957135-700dedea698c?w=800', variantKind: 'jacket', isBestSeller: true, tags: ['fleece'] }),
  buildProduct({ name: 'Cascade Rain Poncho', category: 'cat-outerwear', brand: 'brand-northfield', price: 4500, shortDescription: 'Full-coverage poncho, fits over a pack.', description: 'Oversized enough to cover you and a loaded pack. Not glamorous, extremely effective in a downpour.', imageUrl: 'https://images.unsplash.com/photo-1515372039744-b8f02a3ae446?w=800', stock: 3, tags: ['rain'] }),

  buildProduct({ name: 'Traverse Trail Runner', category: 'cat-footwear', brand: 'brand-fieldstone', price: 14500, shortDescription: 'Responsive cushioning for technical singletrack.', description: 'A rock plate and sticky rubber outsole give this trail runner grip on wet roots and loose scree alike.', imageUrl: 'https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=800', variantKind: 'shoe', isFeatured: true, isBestSeller: true, tags: ['trail-running'] }),
  buildProduct({ name: 'Granite Hiking Boot', category: 'cat-footwear', brand: 'brand-fieldstone', price: 18900, compareAtPrice: 21900, shortDescription: 'Full-grain leather boot with a Vibram sole.', description: 'A resoleable, full-grain leather boot built for multi-day backpacking. Breaks in fast, lasts for years.', imageUrl: 'https://images.unsplash.com/photo-1520639888713-7851133b1ed0?w=800', variantKind: 'shoe', isNewArrival: true, tags: ['hiking', 'leather'] }),
  buildProduct({ name: 'Lowland Camp Slide', category: 'cat-footwear', brand: 'brand-fieldstone', price: 4900, shortDescription: 'Lightweight slide for around camp.', description: 'The shoe you change into after a long day. Closed-cell foam footbed, dries overnight.', imageUrl: 'https://images.unsplash.com/photo-1603487742131-4160ec999306?w=800', variantKind: 'shoe', tags: ['camp'] }),
  buildProduct({ name: 'Meridian Everyday Sneaker', category: 'cat-footwear', brand: 'brand-fieldstone', price: 9900, shortDescription: 'A clean, minimal sneaker for daily wear.', description: 'No logos, no fuss -- a leather-and-canvas sneaker that goes from city streets to easy trails.', imageUrl: 'https://images.unsplash.com/photo-1595950653106-6c9ebd614d3a?w=800', variantKind: 'shoe', tags: ['everyday'] }),

  buildProduct({ name: 'Overland 32L Daypack', category: 'cat-bags', brand: 'brand-rivet', price: 12900, shortDescription: 'A do-everything daypack with a laptop sleeve.', description: 'Structured enough for the office, tough enough for a day hike. Padded 16" laptop sleeve, external water bottle pockets.', imageUrl: 'https://images.unsplash.com/photo-1553062407-98eeb64c6a62?w=800', isFeatured: true, isBestSeller: true, tags: ['daypack'] }),
  buildProduct({ name: 'Voyager Weekender Duffel', category: 'cat-bags', brand: 'brand-rivet', price: 15900, compareAtPrice: 18900, shortDescription: 'Waxed canvas duffel for short trips.', description: 'A waxed-canvas duffel with full-grain leather straps that only looks better with wear.', imageUrl: 'https://images.unsplash.com/photo-1547949003-9792a18a2645?w=800', isNewArrival: true, tags: ['travel'] }),
  buildProduct({ name: 'Alpine 55L Travel Pack', category: 'cat-bags', brand: 'brand-rivet', price: 21900, shortDescription: 'Carry-on-sized pack that opens like a suitcase.', description: 'A front-loading 55L pack sized for carry-on travel, with a stowable harness for checked-bag trips.', imageUrl: 'https://images.unsplash.com/photo-1622560480605-d83c853bc5c3?w=800', tags: ['travel', 'carry-on'] }),
  buildProduct({ name: 'Field Sling Bag', category: 'cat-bags', brand: 'brand-rivet', price: 5900, shortDescription: 'A compact crossbody for essentials.', description: 'Just enough room for a wallet, phone, and a rain shell. The bag you grab for a quick errand.', imageUrl: 'https://images.unsplash.com/photo-1548863227-3af567fc3b27?w=800', stock: 4, tags: ['everyday-carry'] }),

  buildProduct({ name: 'Alpine Merino Tee', category: 'cat-apparel', brand: 'brand-solstice', price: 4500, shortDescription: 'Odor-resistant merino for multi-day wear.', description: 'Merino wool regulates temperature and resists odor, so it is the one shirt in your pack you can wear three days straight.', imageUrl: 'https://images.unsplash.com/photo-1521572163474-6864f9cf17ab?w=800', variantKind: 'tee', isBestSeller: true, tags: ['merino', 'base-layer'] }),
  buildProduct({ name: 'Ridge Flannel Shirt', category: 'cat-apparel', brand: 'brand-solstice', price: 6500, shortDescription: 'Brushed flannel for cool mornings.', description: 'A midweight brushed flannel that layers well under a shell and looks right at the trailhead or in town.', imageUrl: 'https://images.unsplash.com/photo-1521572163474-6864f9cf17ab?w=800', variantKind: 'tee', tags: ['flannel'] }),
  buildProduct({ name: 'Contour Hiking Pant', category: 'cat-apparel', brand: 'brand-solstice', price: 8900, compareAtPrice: 10500, shortDescription: 'Stretch-woven pants with articulated knees.', description: 'A four-way-stretch weave and gusseted crotch mean full range of motion on scrambles and stream crossings.', imageUrl: 'https://images.unsplash.com/photo-1473966968600-fa801b869a1a?w=800', variantKind: 'jacket', isNewArrival: true, tags: ['pants'] }),
  buildProduct({ name: 'Basecamp Thermal Leggings', category: 'cat-apparel', brand: 'brand-solstice', price: 5500, shortDescription: 'Brushed base-layer leggings.', description: 'A brushed interior traps warmth without bulk under your shell pants.', imageUrl: 'https://images.unsplash.com/photo-1483721310020-03333e577078?w=800', variantKind: 'tee', tags: ['base-layer'] }),
  buildProduct({ name: 'Meridian Everyday Tee', category: 'cat-apparel', brand: 'brand-solstice', price: 2900, shortDescription: 'A soft, heavyweight cotton tee.', description: 'Heavyweight cotton with a boxy, relaxed cut -- the tee you reach for every day, not just on the trail.', imageUrl: 'https://images.unsplash.com/photo-1521572163474-6864f9cf17ab?w=800', variantKind: 'tee', tags: ['everyday'] }),

  buildProduct({ name: 'Highline Wool Beanie', category: 'cat-accessories', brand: 'brand-cairn', price: 2900, shortDescription: 'Ribbed wool beanie, unlined.', description: 'A classic ribbed beanie in 100% merino wool. One size, folds to fit most heads.', imageUrl: 'https://images.unsplash.com/photo-1576871337622-98d48d1cf531?w=800', isBestSeller: true, tags: ['hat'] }),
  buildProduct({ name: 'Traverse Liner Gloves', category: 'cat-accessories', brand: 'brand-cairn', price: 2500, shortDescription: 'Touchscreen-compatible liner gloves.', description: 'Thin enough to layer under a mitten, warm enough to wear alone on a brisk walk. Touchscreen-compatible fingertips.', imageUrl: 'https://images.unsplash.com/photo-1610902185448-f4f8fda9e7e2?w=800', tags: ['gloves'] }),
  buildProduct({ name: 'Fieldstone Leather Belt', category: 'cat-accessories', brand: 'brand-rivet', price: 4500, shortDescription: 'Full-grain leather belt, solid brass buckle.', description: 'Cut from a single piece of full-grain leather with a solid brass roller buckle that will outlast the belt itself.', imageUrl: 'https://images.unsplash.com/photo-1624222247344-550fb60583dc?w=800', tags: ['leather'] }),
  buildProduct({ name: 'Ridgeline Polarized Sunglasses', category: 'cat-accessories', brand: 'brand-aerowave', price: 6900, compareAtPrice: 8900, shortDescription: 'Polarized lenses, TR90 frame.', description: 'A lightweight TR90 frame with polarized lenses that cut glare off water and snow.', imageUrl: 'https://images.unsplash.com/photo-1572635196237-14b3f281503f?w=800', isNewArrival: true, tags: ['sunglasses'] }),

  buildProduct({ name: 'Basecamp Steel Mug, 12oz', category: 'cat-home', brand: 'brand-cairn', price: 1900, shortDescription: 'Double-wall insulated camp mug.', description: 'Keeps coffee hot for hours and fits most cup holders. Powder-coated so it never sweats.', imageUrl: 'https://images.unsplash.com/photo-1517663029056-c98a1e35f38f?w=800', isBestSeller: true, tags: ['drinkware'] }),
  buildProduct({ name: 'Overland Cotton Throw Blanket', category: 'cat-home', brand: 'brand-cairn', price: 5900, shortDescription: 'Woven cotton throw, camp-fire ready.', description: 'A heavyweight woven cotton throw big enough for two, machine washable after the inevitable campfire smell.', imageUrl: 'https://images.unsplash.com/photo-1600369672770-985fd30004eb?w=800', tags: ['home'] }),
  buildProduct({ name: 'Field Kitchen Cutting Board', category: 'cat-home', brand: 'brand-cairn', price: 3900, shortDescription: 'End-grain acacia cutting board.', description: 'End-grain construction is gentler on knife edges and more forgiving of daily use than a flat-grain board.', imageUrl: 'https://images.unsplash.com/photo-1594226801341-1d3c5d8f8e37?w=800', stock: 0, tags: ['kitchen'] }),
  buildProduct({ name: 'Solstice Enamel Camp Bowl Set', category: 'cat-home', brand: 'brand-cairn', price: 3400, shortDescription: 'Set of two enamel camp bowls.', description: 'Chip-resistant enamel over steel -- two bowls that go straight from the fire to the table.', imageUrl: 'https://images.unsplash.com/photo-1550583724-b2692b85b150?w=800', tags: ['kitchen'] }),

  buildProduct({ name: 'Northstar Rechargeable Headlamp', category: 'cat-electronics', brand: 'brand-aerowave', price: 4900, shortDescription: '400-lumen USB-C rechargeable headlamp.', description: 'A 400-lumen beam with a red night-vision mode, charges fully in 90 minutes over USB-C.', imageUrl: 'https://images.unsplash.com/photo-1518770660439-4636190af475?w=800', isFeatured: true, isBestSeller: true, tags: ['lighting'], kind: 'simple' }),
  buildProduct({ name: 'Waypoint GPS Tracker', category: 'cat-electronics', brand: 'brand-aerowave', price: 12900, shortDescription: 'Satellite check-in and SOS, no cell signal needed.', description: 'Sends check-ins and SOS alerts over satellite where there is no cell coverage. Requires an active subscription plan (sold separately).', imageUrl: 'https://images.unsplash.com/photo-1526772662000-3f88f10405ff?w=800', isNewArrival: true, tags: ['gps'], kind: 'simple' }),
  buildProduct({ name: 'Portable Solar Power Bank, 20000mAh', category: 'cat-electronics', brand: 'brand-aerowave', price: 8900, compareAtPrice: 10900, shortDescription: 'Foldable solar panel + 20000mAh battery.', description: 'A foldable solar panel trickle-charges the built-in 20000mAh battery, enough for 4 full phone charges.', imageUrl: 'https://images.unsplash.com/photo-1509391366360-2e959784a276?w=800', tags: ['power'] }),
  buildProduct({ name: 'Aurora Trail Camera App — 1yr Digital Plan', category: 'cat-electronics', brand: 'brand-aerowave', price: 2900, shortDescription: 'Digital subscription, delivered by email.', description: 'A one-year digital subscription unlocking offline maps and trail-camera cloud sync. Delivered by email, no physical item ships.', imageUrl: 'https://images.unsplash.com/photo-1526772662000-3f88f10405ff?w=800', kind: 'digital', tags: ['digital', 'subscription'] }),

  buildProduct({ name: 'Trailhead Adjustable Trekking Poles', category: 'cat-fitness', brand: 'brand-fieldstone', price: 6900, shortDescription: 'Carbon-fiber poles, pair.', description: 'Carbon-fiber shafts with cork grips and quick-lock adjustment, sold as a pair.', imageUrl: 'https://images.unsplash.com/photo-1551632811-561732d1e306?w=800', isBestSeller: true, tags: ['poles'] }),
  buildProduct({ name: 'Resistance Band Training Set', category: 'cat-fitness', brand: 'brand-solstice', price: 3400, shortDescription: 'Set of 5 resistance bands + travel bag.', description: 'Five resistance levels in one travel bag -- a full-body workout that fits in a daypack.', imageUrl: 'https://images.unsplash.com/photo-1517836357463-d25dfeac3438?w=800', tags: ['training'] }),
  buildProduct({ name: 'Cairn Cork Yoga Mat', category: 'cat-fitness', brand: 'brand-cairn', price: 5900, shortDescription: 'Natural cork mat with rubber base.', description: 'Cork gets grippier as you sweat, not slipperier. A natural-rubber base keeps it planted on any floor.', imageUrl: 'https://images.unsplash.com/photo-1592432678016-e910b452f9a2?w=800', isNewArrival: true, tags: ['yoga'] }),
]

// -- Reviews -----------------------------------------------------------
const REVIEW_TEMPLATES: { rating: 1 | 2 | 3 | 4 | 5; title: string; body: string }[] = [
  { rating: 5, title: 'Exactly what I needed', body: 'Used this on a four-day trip and it held up perfectly. Would buy again.' },
  { rating: 5, title: 'Great quality', body: 'Fit true to size and the materials feel like they will last for years.' },
  { rating: 4, title: 'Solid, one small gripe', body: 'Really happy with this overall -- only wish it came in more colors.' },
  { rating: 4, title: 'Does the job', body: 'Nothing flashy, just works well day after day.' },
  { rating: 3, title: 'It is fine', body: 'Does what it says, though I expected slightly better finishing on the seams.' },
]

function seededReviews(productId: string, count: number): Review[] {
  const authors = ['Jordan M.', 'Priya S.', 'Sam T.', 'Alex R.', 'Casey W.', 'Morgan L.']
  return Array.from({ length: count }, (_, i) => {
    const t = REVIEW_TEMPLATES[i % REVIEW_TEMPLATES.length]
    return {
      id: `${productId}-rev-${i}`, productId, author: authors[i % authors.length], rating: t.rating,
      title: t.title, body: t.body, verifiedPurchase: i % 3 !== 0,
      helpfulCount: (i * 3) % 21, notHelpfulCount: i % 3,
      createdAt: new Date(2026, (i % 8), 1 + i).toISOString(),
    }
  })
}

// Not every product has reviews -- spec explicitly wants both states.
export const REVIEWS: Review[] = PRODUCTS.flatMap((p, i) => (i % 4 === 0 ? [] : seededReviews(p.id, 2 + (i % 5))))

// Backfill each product's rating/reviewCount from its seeded reviews.
for (const product of PRODUCTS) {
  const productReviews = REVIEWS.filter(r => r.productId === product.id)
  product.reviewCount = productReviews.length
  product.rating = productReviews.length > 0
    ? Math.round((productReviews.reduce((sum, r) => sum + r.rating, 0) / productReviews.length) * 10) / 10
    : 0
}

export const QUESTIONS: Question[] = [
  { id: 'q-1', productId: PRODUCTS[0].id, author: 'Taylor K.', question: 'Is this seam-sealed at every seam or just critical ones?', answer: 'Fully seam-sealed, including the pocket zippers.', answeredBy: 'Aurora Support', createdAt: '2026-03-01T00:00:00Z' },
  { id: 'q-2', productId: PRODUCTS[5].id, author: 'Riley P.', question: 'Do these run true to size or should I size up?', answer: 'True to size for most people; size up half a size if you wear thick wool socks.', answeredBy: 'Aurora Support', createdAt: '2026-03-05T00:00:00Z' },
]

// -- Cross-sell wiring ---------------------------------------------------
for (const product of PRODUCTS) {
  const sameCategory = PRODUCTS.filter(p => p.id !== product.id && p.categoryIds[0] === product.categoryIds[0])
  product.relatedProductIds = sameCategory.slice(0, 4).map(p => p.id)
  product.crossSellProductIds = sameCategory.slice(4, 6).map(p => p.id)
  const otherCategory = PRODUCTS.filter(p => p.categoryIds[0] !== product.categoryIds[0])
  product.upsellProductIds = sameCategory.filter(p => p.price > product.price).slice(0, 2).map(p => p.id)
  product.frequentlyBoughtWithIds = otherCategory.slice(0, 2).map(p => p.id)
}

// -- B2B wholesale terms ---------------------------------------------------
// A handful of realistically-bulk-purchased products (tees, beanies,
// mugs, headlamps, poles) get an MOQ + quantity-break ladder -- most of
// the catalog stays consumer-only (no minOrderQuantity/quantityBreaks at
// all), matching how a real merchant would only wholesale part of a line.
const WHOLESALE_TERMS: Record<string, { minOrderQuantity: number; quantityBreaks: { minQuantity: number; discountPercent: number }[] }> = {
  'Meridian Everyday Tee': { minOrderQuantity: 12, quantityBreaks: [{ minQuantity: 12, discountPercent: 10 }, { minQuantity: 50, discountPercent: 20 }, { minQuantity: 100, discountPercent: 28 }] },
  'Highline Wool Beanie': { minOrderQuantity: 24, quantityBreaks: [{ minQuantity: 24, discountPercent: 12 }, { minQuantity: 75, discountPercent: 22 }] },
  'Basecamp Steel Mug, 12oz': { minOrderQuantity: 24, quantityBreaks: [{ minQuantity: 24, discountPercent: 15 }, { minQuantity: 100, discountPercent: 25 }] },
  'Northstar Rechargeable Headlamp': { minOrderQuantity: 10, quantityBreaks: [{ minQuantity: 10, discountPercent: 8 }, { minQuantity: 30, discountPercent: 16 }] },
  'Trailhead Adjustable Trekking Poles': { minOrderQuantity: 6, quantityBreaks: [{ minQuantity: 6, discountPercent: 10 }, { minQuantity: 20, discountPercent: 18 }] },
}
for (const product of PRODUCTS) {
  const terms = WHOLESALE_TERMS[product.name]
  if (terms) {
    product.minOrderQuantity = terms.minOrderQuantity
    product.quantityBreaks = terms.quantityBreaks
  }
}

// -- Coupons ---------------------------------------------------------------
export const COUPONS: Coupon[] = [
  { code: 'WELCOME10', kind: 'percentage', label: '10% off your first order', value: 10 },
  { code: 'FREESHIP', kind: 'free_shipping', label: 'Free shipping, any order' },
  { code: 'SAVE20', kind: 'fixed', label: '$20 off orders over $150', value: 2000, minSubtotal: 15000 },
  { code: 'BOGOTEE', kind: 'buy_x_get_y', label: 'Buy 2 tees, get 1 free', buyQuantity: 2, getQuantity: 1 },
]

export const FLASH_SALES: FlashSale[] = [
  {
    id: 'flash-1', title: 'End of Season Outerwear Sale',
    productIds: PRODUCTS.filter(p => p.categoryIds[0] === 'cat-outerwear').slice(0, 4).map(p => p.id),
    discountPercent: 25,
    endsAt: new Date(Date.now() + 1000 * 60 * 60 * 30).toISOString(),
  },
]

// -- B2B ------------------------------------------------------------------
// A company's price list is a flat discount off list price (see
// lib/b2bPricing.ts). The demo customer belongs to Trailhead Outfitters as
// its owner, so logging in as morgan@example.com is enough to see every
// B2B surface (company account view, price-list pricing, quick order)
// without a separate demo login.
export const PRICE_LISTS: PriceList[] = [
  { id: 'pricelist-wholesale', name: 'Wholesale', discountPercent: 20 },
  { id: 'pricelist-vip', name: 'VIP Partner', discountPercent: 30 },
]

export const DEMO_COMPANY: Company = {
  id: 'company-1', name: 'Trailhead Outfitters LLC', vatNumber: 'US-84-1234567',
  priceListId: 'pricelist-wholesale', purchaseLimitMinorUnits: 500000, requiresApprovalAboveMinorUnits: 150000,
}

export const DEMO_COMPANY_USERS: CompanyUserSummary[] = [
  { id: 'cu-1', name: 'Morgan Lee', email: 'morgan@example.com', role: 'owner' },
  { id: 'cu-2', name: 'Priya Shah', email: 'priya@trailheadoutfitters.example', role: 'buyer' },
  { id: 'cu-3', name: 'Sam Torres', email: 'sam@trailheadoutfitters.example', role: 'approver' },
  { id: 'cu-4', name: 'Casey Nguyen', email: 'casey@trailheadoutfitters.example', role: 'buyer' },
]

// -- Customer + orders -------------------------------------------------
export const DEMO_CUSTOMER: Customer = {
  id: 'cust-demo', firstName: 'Morgan', lastName: 'Lee', email: 'morgan@example.com', phone: '+1 555-0148',
  addresses: [
    { id: 'addr-1', label: 'Home', firstName: 'Morgan', lastName: 'Lee', line1: '482 Ridgeview Ave', city: 'Boulder', province: 'CO', postalCode: '80302', country: 'US', phone: '+1 555-0148', isDefaultBilling: true, isDefaultShipping: true },
    { id: 'addr-2', label: 'Office', firstName: 'Morgan', lastName: 'Lee', line1: '900 Pearl St, Suite 4', city: 'Boulder', province: 'CO', postalCode: '80302', country: 'US' },
  ],
  preferences: { orderUpdates: true, newsletter: true, productRestock: false },
  company: DEMO_COMPANY,
  companyRole: 'owner',
  createdAt: '2025-11-02T00:00:00Z',
}

export const DEMO_PASSWORD = 'demo-password'

function orderLinesFrom(products: Product[]): Order['lines'] {
  return products.map(p => ({
    productId: p.id, name: p.name, sku: p.sku, imageUrl: p.images[0]?.url, price: p.price, quantity: 1,
  }))
}

export const ORDERS: Order[] = [
  {
    id: 'ord-1', orderNumber: 'AU-10231', customerId: DEMO_CUSTOMER.id,
    lines: orderLinesFrom([PRODUCTS[0], PRODUCTS[9]]),
    subtotal: PRODUCTS[0].price + PRODUCTS[9].price, discount: 0, shipping: 0, tax: 1890,
    total: PRODUCTS[0].price + PRODUCTS[9].price + 1890,
    currency: 'USD', status: 'delivered', paymentStatus: 'paid', paymentMethod: 'card', shippingMethod: 'standard',
    billingAddress: DEMO_CUSTOMER.addresses[0], shippingAddress: DEMO_CUSTOMER.addresses[0],
    trackingNumber: '1Z999AA10123456784',
    trackingEvents: [
      { status: 'Order placed', occurredAt: '2026-01-14T10:00:00Z' },
      { status: 'Shipped', location: 'Denver, CO', occurredAt: '2026-01-15T14:00:00Z' },
      { status: 'Delivered', location: 'Boulder, CO', occurredAt: '2026-01-17T11:20:00Z' },
    ],
    estimatedDeliveryLabel: 'Delivered Jan 17', placedAt: '2026-01-14T10:00:00Z',
  },
  {
    id: 'ord-2', orderNumber: 'AU-10298', customerId: DEMO_CUSTOMER.id,
    lines: orderLinesFrom([PRODUCTS[24]]),
    subtotal: PRODUCTS[24].price, discount: 0, shipping: 900, tax: 380, total: PRODUCTS[24].price + 900 + 380,
    currency: 'USD', status: 'shipped', paymentStatus: 'paid', paymentMethod: 'paypal', shippingMethod: 'express',
    billingAddress: DEMO_CUSTOMER.addresses[0], shippingAddress: DEMO_CUSTOMER.addresses[1],
    trackingNumber: '1Z999AA10987654321',
    trackingEvents: [
      { status: 'Order placed', occurredAt: '2026-02-20T09:00:00Z' },
      { status: 'Shipped', location: 'Denver, CO', occurredAt: '2026-02-21T08:00:00Z' },
    ],
    estimatedDeliveryLabel: 'Arriving Feb 24', placedAt: '2026-02-20T09:00:00Z',
  },
  {
    id: 'ord-3', orderNumber: 'AU-10312', customerId: DEMO_CUSTOMER.id,
    lines: orderLinesFrom([PRODUCTS[13], PRODUCTS[17], PRODUCTS[20]]),
    subtotal: PRODUCTS[13].price + PRODUCTS[17].price + PRODUCTS[20].price, discount: 1000, shipping: 0, tax: 1520,
    total: PRODUCTS[13].price + PRODUCTS[17].price + PRODUCTS[20].price - 1000 + 1520,
    currency: 'USD', status: 'processing', paymentStatus: 'paid', paymentMethod: 'card', shippingMethod: 'standard',
    billingAddress: DEMO_CUSTOMER.addresses[0], shippingAddress: DEMO_CUSTOMER.addresses[0],
    trackingEvents: [{ status: 'Order placed', occurredAt: '2026-03-02T16:40:00Z' }],
    estimatedDeliveryLabel: 'Arriving Mar 7', placedAt: '2026-03-02T16:40:00Z',
  },
]

// -- Blog / CMS ----------------------------------------------------------
export const BLOG_POSTS: BlogPost[] = [
  {
    id: 'post-1', slug: 'layering-for-shoulder-season', title: 'A Field Guide to Layering for Shoulder Season',
    excerpt: 'Three layers, no guesswork -- how to stay dry and warm when the forecast can’t decide either.',
    bodyHtml: '<p>Shoulder season means the weather can swing 20 degrees between the trailhead and the summit. The fix is a predictable three-layer system: a moisture-wicking base, an insulating mid, and a waterproof shell you can strip down to whatever the day needs.</p><p>Start with merino wool against skin -- it manages odor as well as moisture, so a single base layer covers a multi-day trip. Add a fleece or light down mid layer that compresses small enough to live in your pack most of the day. Finish with a shell that is waterproof, not just water-resistant, if there is any real rain in the forecast.</p>',
    coverImageUrl: 'https://images.unsplash.com/photo-1551632811-561732d1e306?w=1200', author: 'Aurora Editorial', publishedAt: '2026-02-10T00:00:00Z', tags: ['guides', 'layering'],
  },
  {
    id: 'post-2', slug: 'choosing-a-daypack', title: 'How to Choose a Daypack That Actually Fits',
    excerpt: 'Torso length matters more than liters. Here is how to measure it in thirty seconds.',
    bodyHtml: '<p>Most daypack returns come down to one thing: torso length, not capacity. Before you shop by liters, measure from your C7 vertebra (the bump at the base of your neck) to the top of your hip bones. That number, not your height, determines which size will actually sit right on your shoulders.</p>',
    coverImageUrl: 'https://images.unsplash.com/photo-1553062407-98eeb64c6a62?w=1200', author: 'Aurora Editorial', publishedAt: '2026-01-22T00:00:00Z', tags: ['guides', 'gear'],
  },
]

export const CMS_PAGES: CmsPage[] = [
  {
    slug: 'about', title: 'About Aurora', metaDescription: 'Aurora makes gear for people who spend real time outside.',
    sections: [
      { type: 'rich_text', id: 'about-1', html: '<h1>About Aurora</h1><p>Aurora started in 2019 with one product: a shell jacket that a small crew of guides in Colorado could not stop recommending to their clients. We still design everything the same way -- built for people who use gear every day, not just on vacation.</p>' },
      { type: 'testimonials', id: 'about-2', title: 'What people say', items: [
        { author: 'Jordan M.', quote: 'The only brand where the gear actually matches the marketing.', rating: 5 },
        { author: 'Priya S.', quote: 'Customer service replaced a zipper pull five years after purchase, no questions asked.', rating: 5 },
      ] },
    ],
  },
  {
    slug: 'shipping-returns', title: 'Shipping & Returns',
    sections: [
      { type: 'rich_text', id: 'sr-1', html: '<h1>Shipping & Returns</h1><p>Standard shipping takes 3-5 business days. Returns are accepted within 60 days of delivery in unworn condition.</p>' },
    ],
  },
]

export const FAQ_ITEMS = [
  { question: 'What is your return policy?', answer: 'Unworn items can be returned within 60 days for a full refund.' },
  { question: 'Do you ship internationally?', answer: 'Currently we ship within the US and Canada. International shipping is on our roadmap.' },
  { question: 'How do I track my order?', answer: 'Every order confirmation email includes a tracking link, and tracking is also available from your account.' },
]

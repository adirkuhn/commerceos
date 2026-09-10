import type { PageSection } from '@/types'
import { CATEGORIES, BRANDS, FLASH_SALES } from './demoData'

// The homepage is data, not a bespoke component -- SectionRenderer (see
// components/sections) renders each entry. A real deployment would load
// this array from a CMS/admin tool instead of a static file; the render
// path is identical either way.
export const HOMEPAGE_SECTIONS: PageSection[] = [
  {
    type: 'hero', id: 'home-hero',
    heading: 'Gear for real days outside',
    subheading: 'Technical outerwear, footwear, and everyday essentials built to be used, not just owned.',
    imageUrl: 'https://images.unsplash.com/photo-1551632811-561732d1e306?w=1600',
    ctaLabel: 'Shop new arrivals', ctaHref: '/search?filter=new_arrivals',
  },
  { type: 'flash_sale', id: 'home-flash-sale', flashSaleId: FLASH_SALES[0].id },
  { type: 'product_grid', id: 'home-featured', title: 'Featured products', subtitle: 'Staff picks from across the catalog', filter: 'featured', limit: 8 },
  { type: 'category_grid', id: 'home-categories', title: 'Shop by category', categoryIds: CATEGORIES.slice(0, 4).map(c => c.id) },
  { type: 'product_grid', id: 'home-new-arrivals', title: 'New arrivals', filter: 'new_arrivals', limit: 4 },
  {
    type: 'banner', id: 'home-banner', tone: 'sale',
    heading: 'End of season outerwear sale', body: 'Up to 25% off shells and insulated jackets, while supplies last.',
    ctaLabel: 'Shop the sale', ctaHref: '/category/outerwear',
  },
  { type: 'product_grid', id: 'home-best-sellers', title: 'Best sellers', filter: 'best_sellers', limit: 8 },
  { type: 'brand_strip', id: 'home-brands', brandIds: BRANDS.map(b => b.id) },
  {
    type: 'testimonials', id: 'home-testimonials', title: 'What people say', items: [
      { author: 'Jordan M.', quote: 'Ordered a shell for a trip to Patagonia and it outperformed jackets twice the price.', rating: 5 },
      { author: 'Priya S.', quote: 'The daypack replaced three different bags I used to carry. Perfect for daily commuting.', rating: 5 },
      { author: 'Sam T.', quote: 'Fast shipping and the trail runners fit exactly as described.', rating: 4 },
    ],
  },
  { type: 'newsletter', id: 'home-newsletter', heading: 'Get 10% off your first order', body: 'Sign up for restock alerts, new arrivals, and seasonal sales.' },
]

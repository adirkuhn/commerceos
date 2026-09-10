// Generic page-builder types -- a homepage (or any CMS page) is an array
// of PageSection, rendered by components/sections/SectionRenderer.tsx.
// Adding a new section type means adding one union member + one renderer
// component, never a new page component.

export interface HeroSection {
  type: 'hero'
  id: string
  heading: string
  subheading?: string
  imageUrl: string
  ctaLabel?: string
  ctaHref?: string
}

export interface ProductGridSection {
  type: 'product_grid'
  id: string
  title: string
  subtitle?: string
  productIds?: string[]
  categoryId?: string
  filter?: 'featured' | 'new_arrivals' | 'best_sellers' | 'recommended'
  limit?: number
}

export interface CategoryGridSection {
  type: 'category_grid'
  id: string
  title: string
  categoryIds: string[]
}

export interface BannerSection {
  type: 'banner'
  id: string
  heading: string
  body: string
  imageUrl?: string
  ctaLabel?: string
  ctaHref?: string
  tone: 'default' | 'sale'
}

export interface RichTextSection {
  type: 'rich_text'
  id: string
  html: string
}

export interface ImageSection {
  type: 'image'
  id: string
  imageUrl: string
  alt: string
}

export interface VideoSection {
  type: 'video'
  id: string
  videoUrl: string
  posterUrl?: string
}

export interface TestimonialsSection {
  type: 'testimonials'
  id: string
  title: string
  items: { author: string; quote: string; rating: number }[]
}

export interface NewsletterSection {
  type: 'newsletter'
  id: string
  heading: string
  body: string
}

export interface BrandStripSection {
  type: 'brand_strip'
  id: string
  brandIds: string[]
}

export interface FlashSaleSection {
  type: 'flash_sale'
  id: string
  flashSaleId: string
}

export type PageSection =
  | HeroSection
  | ProductGridSection
  | CategoryGridSection
  | BannerSection
  | RichTextSection
  | ImageSection
  | VideoSection
  | TestimonialsSection
  | NewsletterSection
  | BrandStripSection
  | FlashSaleSection

export interface CmsPage {
  slug: string
  title: string
  metaDescription?: string
  sections: PageSection[]
}

export interface FaqItem {
  question: string
  answer: string
}

export interface BlogPost {
  id: string
  slug: string
  title: string
  excerpt: string
  bodyHtml: string
  coverImageUrl: string
  author: string
  publishedAt: string
  tags: string[]
}

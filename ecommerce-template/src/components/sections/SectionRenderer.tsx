import type { PageSection } from '@/types'
import { HeroSection } from './HeroSection'
import { ProductGridSection } from './ProductGridSection'
import { CategoryGridSection } from './CategoryGridSection'
import { BannerSection } from './BannerSection'
import { RichTextSection } from './RichTextSection'
import { ImageSection, VideoSection, TestimonialsSection, NewsletterSection, BrandStripSection } from './MiscSections'
import { FlashSaleSection } from './FlashSaleSection'
import { pluginRegistry } from '@/plugins'

// Renders a homepage/CMS page from configuration -- one branch per
// built-in PageSection type, plus a fallback to any section type a
// plugin registered via EcommercePlugin.sections (see types/plugin.ts).
// No page component ever hand-assembles these; see pages/HomePage.tsx.
export function SectionRenderer({ section }: { section: PageSection }) {
  switch (section.type) {
    case 'hero': return <HeroSection section={section} />
    case 'product_grid': return <ProductGridSection section={section} />
    case 'category_grid': return <CategoryGridSection section={section} />
    case 'banner': return <BannerSection section={section} />
    case 'rich_text': return <RichTextSection section={section} />
    case 'image': return <ImageSection section={section} />
    case 'video': return <VideoSection section={section} />
    case 'testimonials': return <TestimonialsSection section={section} />
    case 'newsletter': return <NewsletterSection section={section} />
    case 'brand_strip': return <BrandStripSection section={section} />
    case 'flash_sale': return <FlashSaleSection section={section} />
    default: {
      const plugged = pluginRegistry.sections().find(s => s.type === (section as PageSection).type)
      const Component = plugged?.render(section)
      return Component ? <Component /> : null
    }
  }
}

export function SectionList({ sections }: { sections: PageSection[] }) {
  return (
    <div className="space-y-16">
      {sections.map(section => <SectionRenderer key={section.id} section={section} />)}
    </div>
  )
}

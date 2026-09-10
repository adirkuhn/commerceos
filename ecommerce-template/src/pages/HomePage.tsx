import { Container } from '@/components/layout/Container'
import { SectionList } from '@/components/sections/SectionRenderer'
import { Seo } from '@/components/seo/Seo'
import { HOMEPAGE_SECTIONS } from '@/config/homepage'
import { RecentlyViewedRail } from '@/components/ecommerce/RecentlyViewedRail'

export default function HomePage() {
  return (
    <>
      <Seo title="Gear for real days outside" description="Technical outerwear, footwear, and everyday essentials from Aurora." />
      <Container className="space-y-16 py-10">
        <SectionList sections={HOMEPAGE_SECTIONS} />
        <RecentlyViewedRail />
      </Container>
    </>
  )
}

import { Link } from 'react-router-dom'
import type { HeroSection as HeroSectionType } from '@/types'
import { Button } from '@/components/ui/Button'

export function HeroSection({ section }: { section: HeroSectionType }) {
  return (
    <section className="relative flex min-h-[26rem] items-center overflow-hidden rounded-[var(--radius)]">
      <img src={section.imageUrl} alt="" className="absolute inset-0 h-full w-full object-cover" />
      <div className="absolute inset-0 bg-gradient-to-t from-black/70 via-black/20 to-transparent" />
      <div className="relative z-10 max-w-xl px-6 py-16 text-white sm:px-12">
        <h1 className="text-3xl font-bold tracking-tight sm:text-5xl">{section.heading}</h1>
        {section.subheading && <p className="mt-4 text-lg text-white/90">{section.subheading}</p>}
        {section.ctaLabel && section.ctaHref && (
          <Button asChild size="lg" className="mt-6">
            <Link to={section.ctaHref}>{section.ctaLabel}</Link>
          </Button>
        )}
      </div>
    </section>
  )
}

import { Link } from 'react-router-dom'
import type { BannerSection as BannerSectionType } from '@/types'
import { Button } from '@/components/ui/Button'
import { cn } from '@/lib/utils'

export function BannerSection({ section }: { section: BannerSectionType }) {
  return (
    <section
      className={cn(
        'flex flex-col items-center gap-4 rounded-[var(--radius)] p-8 text-center sm:flex-row sm:justify-between sm:text-left',
        section.tone === 'sale' ? 'bg-destructive text-destructive-foreground' : 'bg-accent text-accent-foreground',
      )}
    >
      <div className="flex items-center gap-4">
        {section.imageUrl && <img src={section.imageUrl} alt="" className="hidden h-16 w-16 rounded-[var(--radius)] object-cover sm:block" />}
        <div>
          <h2 className="text-lg font-semibold">{section.heading}</h2>
          <p className="text-sm opacity-90">{section.body}</p>
        </div>
      </div>
      {section.ctaLabel && section.ctaHref && (
        <Button asChild variant={section.tone === 'sale' ? 'secondary' : 'default'}>
          <Link to={section.ctaHref}>{section.ctaLabel}</Link>
        </Button>
      )}
    </section>
  )
}

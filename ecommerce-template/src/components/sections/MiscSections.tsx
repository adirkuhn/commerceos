import { useState } from 'react'
import { Star } from 'lucide-react'
import type { ImageSection as ImageSectionType, VideoSection as VideoSectionType, TestimonialsSection as TestimonialsSectionType, NewsletterSection as NewsletterSectionType, BrandStripSection as BrandStripSectionType } from '@/types'
import { Input } from '@/components/ui/Input'
import { Button } from '@/components/ui/Button'
import { useBrands } from '@/features/categories/hooks'
import { toast } from '@/components/ui/toast'

export function ImageSection({ section }: { section: ImageSectionType }) {
  return <img src={section.imageUrl} alt={section.alt} className="w-full rounded-[var(--radius)] object-cover" />
}

export function VideoSection({ section }: { section: VideoSectionType }) {
  return (
    <video controls poster={section.posterUrl} className="w-full rounded-[var(--radius)]">
      <source src={section.videoUrl} />
    </video>
  )
}

export function TestimonialsSection({ section }: { section: TestimonialsSectionType }) {
  return (
    <section>
      <h2 className="mb-6 text-xl font-semibold tracking-tight sm:text-2xl">{section.title}</h2>
      <div className="grid gap-4 sm:grid-cols-2">
        {section.items.map((item, i) => (
          <blockquote key={i} className="rounded-[var(--radius)] border border-border bg-card p-6">
            <div className="mb-2 flex" aria-hidden="true">
              {Array.from({ length: 5 }, (_, s) => (
                <Star key={s} className={s < item.rating ? 'h-4 w-4 fill-amber-400 text-amber-400' : 'h-4 w-4 text-muted-foreground'} />
              ))}
            </div>
            <p className="text-sm italic text-muted-foreground">&ldquo;{item.quote}&rdquo;</p>
            <footer className="mt-3 text-sm font-medium">{item.author}</footer>
          </blockquote>
        ))}
      </div>
    </section>
  )
}

export function NewsletterSection({ section }: { section: NewsletterSectionType }) {
  const [email, setEmail] = useState('')
  return (
    <section className="rounded-[var(--radius)] bg-secondary p-8 text-center">
      <h2 className="text-xl font-semibold">{section.heading}</h2>
      <p className="mt-1 text-muted-foreground">{section.body}</p>
      <form
        className="mx-auto mt-4 flex max-w-sm gap-2"
        onSubmit={(e) => {
          e.preventDefault()
          toast({ title: "You're subscribed!", description: email, variant: 'success' })
          setEmail('')
        }}
      >
        <Input type="email" required value={email} onChange={(e) => setEmail(e.target.value)} placeholder="Email address" aria-label="Newsletter email" />
        <Button type="submit">Sign up</Button>
      </form>
    </section>
  )
}

export function BrandStripSection({ section }: { section: BrandStripSectionType }) {
  const { data: brands = [] } = useBrands()
  const items = brands.filter(b => section.brandIds.includes(b.id))
  return (
    <section className="flex flex-wrap items-center justify-center gap-x-10 gap-y-4 border-y border-border py-8">
      {items.map(b => <span key={b.id} className="text-lg font-semibold text-muted-foreground">{b.name}</span>)}
    </section>
  )
}

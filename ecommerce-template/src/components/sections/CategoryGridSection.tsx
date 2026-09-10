import { Link } from 'react-router-dom'
import type { CategoryGridSection as CategoryGridSectionType } from '@/types'
import { useCategories } from '@/features/categories/hooks'
import { Skeleton } from '@/components/ui/Skeleton'

export function CategoryGridSection({ section }: { section: CategoryGridSectionType }) {
  const { data: categories = [], isLoading } = useCategories()
  const items = categories.filter(c => section.categoryIds.includes(c.id))

  return (
    <section>
      <h2 className="mb-6 text-xl font-semibold tracking-tight sm:text-2xl">{section.title}</h2>
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        {isLoading
          ? Array.from({ length: 4 }, (_, i) => <Skeleton key={i} className="aspect-[4/3] w-full" />)
          : items.map(c => (
            <Link key={c.id} to={`/category/${c.slug}`} className="group relative aspect-[4/3] overflow-hidden rounded-[var(--radius)]">
              <img src={c.imageUrl} alt="" className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-105" />
              <div className="absolute inset-0 bg-black/30" />
              <p className="absolute bottom-3 left-3 text-sm font-semibold text-white">{c.name}</p>
            </Link>
          ))}
      </div>
    </section>
  )
}

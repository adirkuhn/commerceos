import { useState } from 'react'
import type { ProductImage } from '@/types'
import { cn } from '@/lib/utils'

// Zoom is a simple CSS-transform-on-hover approach -- no extra
// dependency, works on desktop pointer input; touch devices fall back to
// tap-to-select-thumbnail only, per spec's mobile-specific-UX guidance
// rather than trying to force a desktop zoom interaction onto touch.
export function ProductGallery({ images, alt }: { images: ProductImage[]; alt: string }) {
  const [activeIndex, setActiveIndex] = useState(0)
  const [zoomPos, setZoomPos] = useState({ x: 50, y: 50 })
  const [zooming, setZooming] = useState(false)
  const active = images[activeIndex] ?? images[0]

  if (!active) {
    return <div className="aspect-square w-full rounded-[var(--radius)] bg-muted" />
  }

  return (
    <div className="space-y-3">
      <div
        className="relative aspect-square w-full overflow-hidden rounded-[var(--radius)] bg-muted"
        onMouseMove={(e) => {
          const rect = e.currentTarget.getBoundingClientRect()
          setZoomPos({ x: ((e.clientX - rect.left) / rect.width) * 100, y: ((e.clientY - rect.top) / rect.height) * 100 })
        }}
        onMouseEnter={() => setZooming(true)}
        onMouseLeave={() => setZooming(false)}
      >
        <img
          src={active.url}
          alt={active.alt || alt}
          className="h-full w-full object-cover transition-transform duration-200"
          style={zooming ? { transform: 'scale(1.8)', transformOrigin: `${zoomPos.x}% ${zoomPos.y}%` } : undefined}
        />
      </div>
      {images.length > 1 && (
        <div className="flex gap-2 overflow-x-auto">
          {images.map((img, i) => (
            <button
              key={img.id}
              type="button"
              onClick={() => setActiveIndex(i)}
              aria-label={`Show image ${i + 1} of ${images.length}`}
              aria-current={i === activeIndex}
              className={cn('h-16 w-16 shrink-0 overflow-hidden rounded-[calc(var(--radius)-2px)] border-2', i === activeIndex ? 'border-primary' : 'border-transparent')}
            >
              <img src={img.url} alt="" className="h-full w-full object-cover" />
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

import { useTranslation } from 'react-i18next'
import { X } from 'lucide-react'
import clsx from 'clsx'
import type { Media } from './types'

interface VariantImagePickerProps {
  media: Media[]
  selectedId: string | null | undefined
  onSelect: (mediaId: string | null) => void
}

// Thumbnail grid of the product's images for choosing a variant's image,
// with a leading "No image" choice.
export function VariantImagePicker({ media, selectedId, onSelect }: VariantImagePickerProps) {
  const { t } = useTranslation('pms')
  return (
    <div className="flex flex-wrap gap-1.5 mt-2 p-2 rounded-lg bg-sunken border border-line">
      <button
        type="button"
        onClick={() => onSelect(null)}
        aria-label={t('variant.noImage')}
        className={clsx(
          'w-8 h-8 rounded-md border flex items-center justify-center text-2xs text-ink-4 shrink-0',
          !selectedId ? 'border-accent' : 'border-line',
        )}
      >
        <X size={12} />
      </button>
      {media.map(m => (
        <button aria-pressed={m.id === selectedId}
          key={m.id}
          type="button"
          onClick={() => onSelect(m.id)}
          aria-label={m.alt_text || t('variant.productImage')}
          className={clsx(
            'w-8 h-8 rounded-md overflow-hidden border shrink-0',
            m.id === selectedId ? 'border-accent' : 'border-line',
          )}
        >
          <img src={m.url} alt={m.alt_text} className="w-full h-full object-cover" />
        </button>
      ))}
    </div>
  )
}

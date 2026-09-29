import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { X } from 'lucide-react'
import clsx from 'clsx'
import { FormCard } from './FormCard'
import { inputClass, labelClass } from './formStyles'
import type { ProductFields, SetProductField } from './productFields'

interface ProductDetailsCardProps {
  fields: ProductFields
  setField: SetProductField
  // Slug is only editable once the product exists (create derives it).
  showSlug: boolean
}

// The "General" card: name, slug, description, tags.
export function ProductDetailsCard({ fields, setField, showSlug }: ProductDetailsCardProps) {
  const { t } = useTranslation('pms')
  return (
    <FormCard title={t('details.title')}>
      <div>
        <label htmlFor="product-name" className={labelClass}>{t('details.name')}</label>
        <input
          id="product-name"
          type="text" required value={fields.name} onChange={e => setField('name', e.target.value)}
          className={inputClass}
        />
      </div>
      {showSlug && (
        <div>
          <label htmlFor="product-slug" className={labelClass}>{t('details.slug')}</label>
          <input
            id="product-slug"
            type="text" value={fields.slug} onChange={e => setField('slug', e.target.value)}
            pattern="^[a-z0-9]+(-[a-z0-9]+)*$"
            title={t('details.slugHint')}
            className={inputClass}
          />
        </div>
      )}
      <div>
        <label htmlFor="product-description" className={labelClass}>{t('details.description')}</label>
        <textarea
          id="product-description"
          value={fields.description} onChange={e => setField('description', e.target.value)} rows={3}
          className={inputClass}
        />
      </div>
      <TagsField tags={fields.tags} onChange={tags => setField('tags', tags)} />
    </FormCard>
  )
}

// Chip-style tag entry: Enter/comma adds, Backspace on an empty input
// removes the last tag, blur commits whatever was typed.
export function TagsField({ tags, onChange }: { tags: string[]; onChange: (tags: string[]) => void }) {
  const { t } = useTranslation('pms')
  const [tagInput, setTagInput] = useState('')

  const addTag = (raw: string) => {
    const value = raw.trim()
    if (value && !tags.includes(value)) onChange([...tags, value])
    setTagInput('')
  }
  const removeTag = (value: string) => onChange(tags.filter(tag => tag !== value))
  const handleTagInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault()
      addTag(tagInput)
    } else if (e.key === 'Backspace' && !tagInput && tags.length > 0) {
      removeTag(tags[tags.length - 1])
    }
  }

  return (
    <div>
      <label htmlFor="product-tags" className={labelClass}>{t('details.tags')}</label>
      <div className={clsx(inputClass, 'flex flex-wrap items-center gap-1.5 h-auto min-h-[2.25rem] py-1.5')}>
        {tags.map(tag => (
          <span
            key={tag}
            className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-2xs bg-hover text-ink-2"
          >
            {tag}
            <button type="button" onClick={() => removeTag(tag)} aria-label={t('details.removeTag', { tag })}>
              <X size={10} className="text-ink-4" />
            </button>
          </span>
        ))}
        <input
          id="product-tags"
          type="text"
          value={tagInput}
          onChange={e => setTagInput(e.target.value)}
          onKeyDown={handleTagInputKeyDown}
          onBlur={() => addTag(tagInput)}
          placeholder={tags.length === 0 ? t('details.tagsPlaceholder') : ''}
          className="flex-1 min-w-[6rem] bg-transparent outline-none focus-visible:ring-2 focus-visible:ring-focus/60 text-xs text-ink-2"
        />
      </div>
      <p className="mt-1.5 text-2xs text-ink-4">
        {t('details.tagsHint')}
      </p>
    </div>
  )
}

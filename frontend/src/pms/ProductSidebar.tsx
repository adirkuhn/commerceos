import { useState } from 'react'
import clsx from 'clsx'
import { FormCard } from './FormCard'
import { CategoryMarginEditor } from './ProductPricingCards'
import { sortCategoriesHierarchically } from './categoryTree'
import { inputClass, labelClass } from './formStyles'
import type { Category, ProductStatus } from './types'

interface ProductSidebarProps {
  status: ProductStatus
  onStatusChange: (status: ProductStatus) => void
  categoryId: string
  onCategoryChange: (categoryId: string) => void
  categories: Category[]
  // Resolves true once the category exists (and was selected), so the
  // "new category" input can be cleared; false leaves it for a retry.
  onAddCategory: (name: string) => Promise<boolean>
  onUpdateCategoryMargin: (categoryId: string, margin: number | null) => Promise<Category>
}

// Right-hand column of the General tab: status, category (with inline
// create + target margin), and the not-yet-supported placeholders.
export function ProductSidebar({ status, onStatusChange, categoryId, onCategoryChange, categories, onAddCategory, onUpdateCategoryMargin }: ProductSidebarProps) {
  const [newCategoryName, setNewCategoryName] = useState('')

  const handleAddCategory = async () => {
    if (!newCategoryName.trim()) return
    if (await onAddCategory(newCategoryName.trim())) setNewCategoryName('')
  }

  return (
    <div className="space-y-5">
      <div className="rounded-xl border border-line p-4 divide-y divide-line/60">
        <div className="pb-4">
          <label htmlFor="product-status-sidebar" className={labelClass}>Status</label>
          <select
            id="product-status-sidebar"
            value={status} onChange={e => onStatusChange(e.target.value as ProductStatus)}
            className={inputClass}
          >
            <option value="draft">Draft</option>
            <option value="active">Active</option>
            <option value="archived">Archived</option>
          </select>
        </div>

        <div className="pt-4 pb-4">
          <label htmlFor="product-category" className={labelClass}>Category</label>
          <select
            id="product-category"
            value={categoryId} onChange={e => onCategoryChange(e.target.value)}
            className={inputClass}
          >
            <option value="">No category</option>
            {sortCategoriesHierarchically(categories).map(c => (
              <option key={c.id} value={c.id}>{'    '.repeat(c.depth)}{c.depth > 0 ? '↳ ' : ''}{c.name}</option>
            ))}
          </select>
          <div className="flex gap-1.5 mt-1.5">
            <input
              type="text" value={newCategoryName} onChange={e => setNewCategoryName(e.target.value)}
              placeholder="+ new category name"
              className="flex-1 px-3 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
            />
            <button type="button" onClick={handleAddCategory} className="px-3 py-1.5 rounded-lg dark:bg-night-800 bg-gray-100 text-2xs font-medium text-ink-2 hover:bg-hover transition-colors">
              Add
            </button>
          </div>
          {categoryId && (
            <CategoryMarginEditor
              category={categories.find(c => c.id === categoryId) ?? null}
              onSave={margin => onUpdateCategoryMargin(categoryId, margin)}
            />
          )}
        </div>

        <div className="pt-4">
          <label htmlFor="product-vendor" className={labelClass}>Vendor</label>
          <input id="product-vendor" disabled placeholder="Coming soon" className={clsx(inputClass, 'cursor-not-allowed opacity-60')} />
        </div>

        <div className="pt-4">
          <label htmlFor="product-collections" className={labelClass}>Collections</label>
          <input id="product-collections" disabled placeholder="Coming soon" className={clsx(inputClass, 'cursor-not-allowed opacity-60')} />
        </div>
      </div>

      <FormCard title="SEO" description="Coming soon.">
        <div>
          <label htmlFor="product-seo-title" className={labelClass}>Page title</label>
          <input id="product-seo-title" disabled placeholder="Coming soon" className={clsx(inputClass, 'cursor-not-allowed opacity-60')} />
        </div>
        <div>
          <label htmlFor="product-seo-description" className={labelClass}>Meta description</label>
          <textarea id="product-seo-description" disabled placeholder="Coming soon" rows={2} className={clsx(inputClass, 'cursor-not-allowed opacity-60')} />
        </div>
      </FormCard>
    </div>
  )
}

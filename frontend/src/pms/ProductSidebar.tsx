import { useState } from 'react'
import { useTranslation } from 'react-i18next'
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
  const { t } = useTranslation('pms')
  const [newCategoryName, setNewCategoryName] = useState('')

  const handleAddCategory = async () => {
    if (!newCategoryName.trim()) return
    if (await onAddCategory(newCategoryName.trim())) setNewCategoryName('')
  }

  return (
    <div className="space-y-5">
      <div className="rounded-xl border border-line p-4 divide-y divide-line/60">
        <div className="pb-4">
          <label htmlFor="product-status-sidebar" className={labelClass}>{t('status.label')}</label>
          <select
            id="product-status-sidebar"
            value={status} onChange={e => onStatusChange(e.target.value as ProductStatus)}
            className={inputClass}
          >
            <option value="draft">{t('status.draft')}</option>
            <option value="active">{t('status.active')}</option>
            <option value="archived">{t('status.archived')}</option>
          </select>
        </div>

        <div className="pt-4 pb-4">
          <label htmlFor="product-category" className={labelClass}>{t('sidebar.category')}</label>
          <select
            id="product-category"
            value={categoryId} onChange={e => onCategoryChange(e.target.value)}
            className={inputClass}
          >
            <option value="">{t('sidebar.noCategory')}</option>
            {sortCategoriesHierarchically(categories).map(c => (
              <option key={c.id} value={c.id}>{'    '.repeat(c.depth)}{c.depth > 0 ? '↳ ' : ''}{c.name}</option>
            ))}
          </select>
          <div className="flex gap-1.5 mt-1.5">
            <input
              type="text" value={newCategoryName} onChange={e => setNewCategoryName(e.target.value)}
              placeholder={t('sidebar.newCategory')}
              className="flex-1 px-3 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
            />
            <button type="button" onClick={handleAddCategory} className="px-3 py-1.5 rounded-lg dark:bg-night-800 bg-gray-100 text-2xs font-medium text-ink-2 hover:bg-hover transition-colors">
              {t('sidebar.add')}
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
          <label htmlFor="product-vendor" className={labelClass}>{t('sidebar.vendor')}</label>
          <input id="product-vendor" disabled placeholder={t('sidebar.comingSoon')} className={clsx(inputClass, 'cursor-not-allowed opacity-60')} />
        </div>

        <div className="pt-4">
          <label htmlFor="product-collections" className={labelClass}>{t('sidebar.collections')}</label>
          <input id="product-collections" disabled placeholder={t('sidebar.comingSoon')} className={clsx(inputClass, 'cursor-not-allowed opacity-60')} />
        </div>
      </div>

      <FormCard title={t('sidebar.seo')} description={t('sidebar.comingSoonDot')}>
        <div>
          <label htmlFor="product-seo-title" className={labelClass}>{t('sidebar.pageTitle')}</label>
          <input id="product-seo-title" disabled placeholder={t('sidebar.comingSoon')} className={clsx(inputClass, 'cursor-not-allowed opacity-60')} />
        </div>
        <div>
          <label htmlFor="product-seo-description" className={labelClass}>{t('sidebar.metaDescription')}</label>
          <textarea id="product-seo-description" disabled placeholder={t('sidebar.comingSoon')} rows={2} className={clsx(inputClass, 'cursor-not-allowed opacity-60')} />
        </div>
      </FormCard>
    </div>
  )
}

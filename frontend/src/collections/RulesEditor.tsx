import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import { Plus, X } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import { listCategories } from '../pms/api'
import type { Category } from '../pms/types'
import { fromMinorUnits, newRule, toMinorUnits } from './form'
import { inputClass } from './styles'
import type { CollectionMatch, CollectionRule, CollectionRuleField } from './types'

const fieldLabels = (t: TFunction<'collections'>): Record<CollectionRuleField, string> => ({
  category: t('rules.category'), tag: t('rules.tag'), price: t('rules.price'), on_sale: t('rules.onSale'), added_within_days: t('rules.addedWithin'),
})

function RuleRow({ rule, categories, onChange, onRemove, index }: {
  rule: CollectionRule; categories: Category[]; onChange: (r: CollectionRule) => void; onRemove: () => void; index: number
}) {
  const { t } = useTranslation('collections')
  const labels = fieldLabels(t)
  const n = index + 1
  const [min, setMin] = useState(fromMinorUnits(rule.min_minor_units))
  const [max, setMax] = useState(fromMinorUnits(rule.max_minor_units))
  return (
    <li className="flex flex-wrap items-center gap-2 rounded-lg border border-line p-2">
      <label className="sr-only" htmlFor={`rule-${index}-field`}>{t('rules.condition', { number: n })}</label>
      <select id={`rule-${index}-field`} className={`${inputClass} w-44`} value={rule.field} onChange={e => onChange(newRule(e.target.value as CollectionRuleField))}>
        {(Object.keys(labels) as CollectionRuleField[]).map(f => <option key={f} value={f}>{labels[f]}</option>)}
      </select>
      {rule.field === 'category' && (
        <>
          <select aria-label={t('rules.conditionCategory', { number: n })} className={`${inputClass} w-48`} value={rule.value ?? ''} onChange={e => onChange({ ...rule, value: e.target.value })}>
            <option value="">{t('rules.chooseCategory')}</option>
            {categories.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
          </select>
          <label className="flex items-center gap-1 text-2xs text-ink-3">
            <input type="checkbox" checked={!!rule.include_subcategories} onChange={e => onChange({ ...rule, include_subcategories: e.target.checked })} />
            {t('rules.subcategories')}
          </label>
        </>
      )}
      {rule.field === 'tag' && (
        <input aria-label={t('rules.conditionTag', { number: n })} className={`${inputClass} w-48`} value={rule.value ?? ''} maxLength={100} onChange={e => onChange({ ...rule, value: e.target.value })} />
      )}
      {rule.field === 'price' && (
        <>
          <input aria-label={t('rules.conditionMin', { number: n })} placeholder={t('rules.min')} inputMode="decimal" className={`${inputClass} w-24`} value={min}
            onChange={e => { setMin(e.target.value); onChange({ ...rule, min_minor_units: toMinorUnits(e.target.value) }) }} />
          <span className="text-2xs text-ink-4">{t('rules.and')}</span>
          <input aria-label={t('rules.conditionMax', { number: n })} placeholder={t('rules.max')} inputMode="decimal" className={`${inputClass} w-24`} value={max}
            onChange={e => { setMax(e.target.value); onChange({ ...rule, max_minor_units: toMinorUnits(e.target.value) }) }} />
        </>
      )}
      {rule.field === 'added_within_days' && (
        <>
          <input aria-label={t('rules.conditionDays', { number: n })} type="number" min={1} max={3650} className={`${inputClass} w-20`} value={rule.days ?? 30}
            onChange={e => onChange({ ...rule, days: Math.max(1, Math.min(3650, Number(e.target.value) || 1)) })} />
          <span className="text-2xs text-ink-4">{t('rules.days')}</span>
        </>
      )}
      <button type="button" aria-label={t('rules.remove', { number: n })} onClick={onRemove} className="ml-auto p-1 text-ink-4 hover:text-danger"><X size={12} /></button>
    </li>
  )
}

/** Smart collection conditions: products matching all (or any) are included. */
export function RulesEditor({ match, rules, onMatch, onRules }: {
  match: CollectionMatch; rules: CollectionRule[]; onMatch: (m: CollectionMatch) => void; onRules: (r: CollectionRule[]) => void
}) {
  const { token } = useAuth()
  const { t } = useTranslation('collections')
  const [categories, setCategories] = useState<Category[]>([])
  useEffect(() => {
    listCategories(token ?? '').then(setCategories, () => setCategories([]))
  }, [token])
  return (
    <div className="space-y-2">
      <p className="text-xs text-ink-2">
        {t('rules.mustMatchBefore')}{' '}
        <select aria-label={t('rules.matchLabel')} className="rounded border border-line bg-raised px-1 text-xs" value={match} onChange={e => onMatch(e.target.value as CollectionMatch)}>
          <option value="all">{t('match.all')}</option>
          <option value="any">{t('match.any')}</option>
        </select>{' '}
        {t('rules.mustMatchAfter')}
      </p>
      <ul className="space-y-2">
        {rules.map((r, i) => (
          <RuleRow key={i} index={i} rule={r} categories={categories}
            onChange={next => onRules(rules.map((x, j) => (j === i ? next : x)))}
            onRemove={() => onRules(rules.filter((_, j) => j !== i))} />
        ))}
      </ul>
      {rules.length < 10 && (
        <button type="button" onClick={() => onRules([...rules, newRule('tag')])} className="flex items-center gap-1 text-2xs text-accent">
          <Plus size={11} /> {t('rules.add')}
        </button>
      )}
    </div>
  )
}

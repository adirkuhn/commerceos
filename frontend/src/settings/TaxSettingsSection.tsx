import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { X, Trash2 } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import { useRoutedEditor, NEW_RECORD } from '../routing/useRoutedEditor'
import { useInitOnce } from '../routing/useInitOnce'
import { useTenantSettings } from './hooks/useTenantSettings'
import { useTaxClasses } from '../tax/hooks/useTaxClasses'
import { useTaxRates } from '../tax/hooks/useTaxRates'
import type { TaxClass } from '../tax/types'
import { Dialog } from '../components/common/Dialog'
import { useConfirm } from '../components/common/useConfirm'
import { DefaultVatClassWarning } from './DefaultVatClassWarning'
import { TAX_CLASSES_PATH } from './settingsPaths'
import { apiErrorText } from '../api/apiErrorText'

interface TaxClassFormState {
  name: string
  isDefault: boolean
}

export function TaxSettingsSection() {
  const { confirm, confirmDialog } = useConfirm()
  const { t } = useTranslation(['settings', 'common'])
  const { token } = useAuth()
  const { settings, save } = useTenantSettings(token ?? '')
  const { classes, loading: classesLoading, error: classesError, create, update, remove } = useTaxClasses(token ?? '')
  const { rates, error: ratesError, upsert, remove: removeRate } = useTaxRates(token ?? '')

  const [pricesIncludeTax, setPricesIncludeTax] = useState(true)
  const [defaultTaxCountry, setDefaultTaxCountry] = useState('')
  const [hydrated, setHydrated] = useState(false)
  const [savingDefaults, setSavingDefaults] = useState(false)
  const [defaultsError, setDefaultsError] = useState('')
  const [defaultsSaved, setDefaultsSaved] = useState(false)

  if (!hydrated && settings) {
    setPricesIncludeTax(settings.prices_include_tax)
    setDefaultTaxCountry(settings.default_tax_country)
    setHydrated(true)
  }

  async function handleSaveDefaults() {
    setSavingDefaults(true)
    setDefaultsError('')
    setDefaultsSaved(false)
    try {
      await save({ prices_include_tax: pricesIncludeTax, default_tax_country: defaultTaxCountry })
      setDefaultsSaved(true)
    } catch (err) {
      setDefaultsError(apiErrorText(err) ?? t('tax.saveDefaultsFailed'))
    } finally {
      setSavingDefaults(false)
    }
  }

  // Add/edit tax class form open state is a URL (/settings/tax/classes/new
  // or /<id>), so it survives a refresh; the form fills once the class list
  // has it.
  const classEditor = useRoutedEditor(TAX_CLASSES_PATH)
  const creatingClass = classEditor.editingId === NEW_RECORD
  const editingClass = classEditor.editingId && !creatingClass
    ? classes.find(c => c.id === classEditor.editingId) ?? null
    : null
  const [classForm, setClassForm] = useState<TaxClassFormState>({ name: '', isDefault: false })
  const [classSaving, setClassSaving] = useState(false)
  const [classFormError, setClassFormError] = useState('')
  const [rowError, setRowError] = useState('')

  const classFormReady = useInitOnce(classEditor.editingId, creatingClass || !!editingClass, () => {
    setClassForm(editingClass ? { name: editingClass.name, isDefault: editingClass.is_default } : { name: '', isDefault: false })
    setClassFormError('')
  })
  const openCreateClass = classEditor.openNew
  const openEditClass = (c: TaxClass) => classEditor.openEdit(c.id)
  const closeClassForm = classEditor.close

  async function handleClassSubmit(e: React.FormEvent) {
    e.preventDefault()
    setClassFormError('')
    setClassSaving(true)
    try {
      if (editingClass) {
        await update(editingClass.id, { name: classForm.name, is_default: classForm.isDefault })
      } else {
        await create({ name: classForm.name, is_default: classForm.isDefault })
      }
      closeClassForm()
    } catch (err) {
      setClassFormError(apiErrorText(err) ?? t('tax.saveClassFailed'))
    } finally {
      setClassSaving(false)
    }
  }

  async function handleDeleteClass(c: TaxClass) {
    if (!(await confirm({ title: t('tax.deleteClassTitle', { name: c.name }), message: t('tax.cannotUndo'), confirmLabel: t('tax.deleteClass'), destructive: true }))) return
    setRowError('')
    try {
      await remove(c.id)
    } catch (err) {
      setRowError(apiErrorText(err) ?? t('tax.deleteClassFailed'))
    }
  }

  async function handleMakeDefault(c: TaxClass) {
    setRowError('')
    try {
      await update(c.id, { name: c.name, is_default: true })
    } catch (err) {
      setRowError(apiErrorText(err) ?? t('tax.saveClassFailed'))
    }
  }

  async function handleAddRate(taxClassId: string, country: string, ratePercent: string) {
    setRowError('')
    try {
      await upsert({ tax_class_id: taxClassId, country, rate_basis_points: Math.round(parseFloat(ratePercent) * 100) })
    } catch (err) {
      setRowError(apiErrorText(err) ?? t('tax.saveRateFailed'))
    }
  }

  async function handleRemoveRate(id: string) {
    setRowError('')
    try {
      await removeRate(id)
    } catch (err) {
      setRowError(apiErrorText(err) ?? t('tax.deleteRateFailed'))
    }
  }

  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <h2 className="text-base font-semibold text-ink mb-1">{t('sections.tax')}</h2>
        <p className="text-xs text-ink-4">{t('tax.subtitle')}</p>
      </div>

      <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
        <h3 className="text-sm font-semibold text-ink">{t('tax.storeDefaults')}</h3>
        <div className="flex items-center justify-between">
          <span className="text-xs text-ink-2">{t('tax.pricesIncludeTax')}</span>
          <label className="relative flex items-center shrink-0 cursor-pointer">
            <input
              type="checkbox"
              role="switch"
              aria-label={t('tax.pricesIncludeTax')}
              checked={pricesIncludeTax}
              onChange={e => { setPricesIncludeTax(e.target.checked); setDefaultsSaved(false) }}
              className="sr-only peer"
            />
            <div className="w-8 h-4 dark:bg-night-700 bg-gray-300 rounded-full peer peer-checked:bg-accent-solid transition-colors" />
            <div className="absolute left-0.5 top-0.5 w-3 h-3 bg-white rounded-full transition-all peer-checked:translate-x-4" />
          </label>
        </div>
        <div className="max-w-[160px]">
          <label htmlFor="default-tax-country" className="block text-2xs text-ink-4 caps-label mb-1.5">
            {t('tax.defaultCountry')}
          </label>
          <input
            id="default-tax-country"
            value={defaultTaxCountry}
            maxLength={2}
            onChange={e => { setDefaultTaxCountry(e.target.value.toUpperCase()); setDefaultsSaved(false) }}
            className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
        {defaultsError && <p className="text-xs text-danger">{defaultsError}</p>}
        {defaultsSaved && <p className="text-xs text-success">{t('savedMessage')}</p>}
        <button
          onClick={handleSaveDefaults}
          disabled={savingDefaults}
          className="px-3 py-1.5 rounded-lg bg-fill text-xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
        >
          {savingDefaults ? t('common:actions.saving') : t('tax.saveDefaults')}
        </button>
      </div>

      <DefaultVatClassWarning classes={classes} rates={rates} country={settings?.default_tax_country ?? defaultTaxCountry} />

      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-ink">{t('tax.classes')}</h3>
        <button
          onClick={openCreateClass}
          className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover transition-colors"
        >
          {t('tax.addClass')}
        </button>
      </div>
      {classesError && <p className="text-xs text-danger">{classesError}</p>}
      {ratesError && <p className="text-xs text-danger">{ratesError}</p>}
      {rowError && <p className="text-xs text-danger">{rowError}</p>}
      {classesLoading && <p className="text-xs text-ink-4">{t('common:status.loading')}</p>}

      <div className="space-y-3">
        {classes.map(c => (
          <div key={c.id} className="bg-raised rounded-xl border border-line p-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium text-ink">{c.name}</span>
                {c.is_default && (
                  <span className="text-2xs px-2 py-0.5 rounded-full  bg-accent/10 text-accent">{t('tax.default')}</span>
                )}
              </div>
              <div className="flex gap-2">
                {!c.is_default && (
                  <button
                    onClick={() => void handleMakeDefault(c)}
                    className="py-1 px-2 rounded-lg bg-fill text-2xs text-ink-3 hover:text-accent transition-colors"
                  >
                    {t('tax.makeDefault')}
                  </button>
                )}
                <button
                  onClick={() => openEditClass(c)}
                  className="py-1 px-2 rounded-lg bg-fill text-2xs text-ink-3 hover:text-accent transition-colors"
                >
                  {t('common:actions.edit')}
                </button>
                <button
                  onClick={() => handleDeleteClass(c)}
                  className="py-1 px-2 rounded-lg bg-fill text-2xs text-ink-3 hover:text-danger transition-colors"
                >
                  {t('common:actions.delete')}
                </button>
              </div>
            </div>
            <div className="mt-3 space-y-1.5">
              {rates.filter(r => r.tax_class_id === c.id).map(r => (
                <div key={r.id} className="flex items-center justify-between text-xs text-ink-2">
                  <span>{r.country}</span>
                  <span>{(r.rate_basis_points / 100).toFixed(2)}%</span>
                  <button
                    onClick={() => handleRemoveRate(r.id)}
                    className="text-2xs text-ink-4 hover:text-danger transition-colors flex items-center gap-1"
                  >
                    <Trash2 size={11} /> {t('tax.removeRate')}
                  </button>
                </div>
              ))}
            </div>
            <TaxRateAddForm taxClassId={c.id} onAdd={handleAddRate} />
          </div>
        ))}
      </div>

      {classFormReady && (
        <Dialog onClose={closeClassForm} className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={closeClassForm} />
          <div className="relative w-full max-w-sm bg-raised rounded-xl shadow-2xl border border-line p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-ink">
                {editingClass ? t('tax.editClass') : t('tax.addClass')}
              </h3>
              <button onClick={closeClassForm} aria-label={t('common:actions.close')}>
                <X size={14} className="text-ink-4" />
              </button>
            </div>
            <form onSubmit={handleClassSubmit} className="space-y-3">
              <div>
                <label htmlFor="tax-class-name" className="block text-2xs text-ink-4 caps-label mb-1.5">
                  {t('common:fields.name')}
                </label>
                <input
                  id="tax-class-name"
                  type="text"
                  required
                  autoFocus
                  value={classForm.name}
                  onChange={e => setClassForm(f => ({ ...f, name: e.target.value }))}
                  className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                />
              </div>
              <div className="flex items-center justify-between">
                <span className="text-2xs text-ink-4 caps-label">{t('tax.defaultClass')}</span>
                <label className="relative flex items-center shrink-0 cursor-pointer">
                  <input
                    type="checkbox"
                    role="switch"
                    aria-label={t('tax.defaultClass')}
                    checked={classForm.isDefault}
                    onChange={e => setClassForm(f => ({ ...f, isDefault: e.target.checked }))}
                    className="sr-only peer"
                  />
                  <div className="w-8 h-4 dark:bg-night-700 bg-gray-300 rounded-full peer peer-checked:bg-accent-solid transition-colors" />
                  <div className="absolute left-0.5 top-0.5 w-3 h-3 bg-white rounded-full transition-all peer-checked:translate-x-4" />
                </label>
              </div>
              {classFormError && (
                <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{classFormError}</p>
              )}
              <button
                type="submit"
                disabled={classSaving}
                className="w-full py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
              >
                {classSaving ? t('common:actions.saving') : editingClass ? t('common:actions.saveChanges') : t('tax.addClassShort')}
              </button>
            </form>
          </div>
        </Dialog>
      )}
      {confirmDialog}
    </div>
  )
}

function TaxRateAddForm({ taxClassId, onAdd }: { taxClassId: string; onAdd: (taxClassId: string, country: string, ratePercent: string) => void }) {
  const { t } = useTranslation('settings')
  const [country, setCountry] = useState('')
  const [ratePercent, setRatePercent] = useState('')

  return (
    <form
      onSubmit={e => {
        e.preventDefault()
        onAdd(taxClassId, country, ratePercent)
        setCountry('')
        setRatePercent('')
      }}
      className="mt-3 pt-3 border-t border-line flex items-end gap-2"
    >
      <div>
        <label htmlFor={`tax-rate-country-${taxClassId}`} className="block text-2xs text-ink-4 caps-label mb-1.5">
          {t('tax.country')}
        </label>
        <input
          id={`tax-rate-country-${taxClassId}`}
          value={country}
          maxLength={2}
          onChange={e => setCountry(e.target.value.toUpperCase())}
          className="w-16 px-2 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
        />
      </div>
      <div>
        <label htmlFor={`tax-rate-percent-${taxClassId}`} className="block text-2xs text-ink-4 caps-label mb-1.5">
          {t('tax.ratePercent')}
        </label>
        <input
          id={`tax-rate-percent-${taxClassId}`}
          type="number"
          step="0.01"
          value={ratePercent}
          onChange={e => setRatePercent(e.target.value)}
          className="w-20 px-2 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
        />
      </div>
      <button
        type="submit"
        className="px-3 py-1.5 rounded-lg bg-fill text-xs font-medium text-ink-2 hover:bg-hover transition-colors"
      >
        {t('tax.addRate')}
      </button>
    </form>
  )
}

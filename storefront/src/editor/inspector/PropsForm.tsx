import { useTranslation } from 'react-i18next'
import type { ComponentDefinition, PropField } from '../../components/types'
import { useEditorLabels } from '../i18n/useEditorLabels'
import type { RichText } from '../../core/richText'
import { LinkField, NumberField, SelectField, TextField, ToggleField } from './fields'
import type { NavLink } from '../../components/site/site'
import { LinksField } from './LinksField'
import { SocialLinksField } from './SocialLinksField'
import type { SocialLink } from '../../components/site/social'
import { AssetPicker, CategorySelect, CollectionSelect, CollectionsPicker, ProductPicker, ProductsPicker } from './pickers'
import { RichTextField } from './RichTextField'
import { isTranslatable } from '../../components/translatable'

type Props = Record<string, unknown>

function FieldFor({ def, name, field, value, onChange }: { def: ComponentDefinition; name: string; field: PropField; value: unknown; onChange: (v: unknown) => void }) {
  const labels = useEditorLabels()
  if (field.kind === 'hidden') return null
  const label = labels.field(def, name, field)
  switch (field.kind) {
    case 'text': return <TextField label={label} value={String(value ?? '')} maxLength={field.maxLength} multiline={field.multiline} onChange={onChange} />
    case 'number': return <NumberField label={label} value={Number(value) || 0} min={field.min} max={field.max} onChange={onChange} />
    case 'select': return <SelectField label={label} value={(value ?? '') as string | number} options={field.options.map(o => ({ value: o.value, label: labels.option(def, name, o) }))} onChange={onChange} />
    case 'toggle': return <ToggleField label={label} value={Boolean(value)} onChange={onChange} />
    case 'richtext': return <RichTextField label={label} value={(value as RichText) ?? []} onChange={onChange} />
    case 'asset': return <AssetPicker label={label} value={String(value ?? '')} onChange={onChange} />
    case 'link': return <LinkField label={label} value={String(value ?? '')} onChange={onChange} />
    case 'category': return <CategorySelect label={label} value={String(value ?? '')} onChange={onChange} />
    case 'collection': return <CollectionSelect label={label} value={String(value ?? '')} onChange={onChange} />
    case 'product': return <ProductPicker label={label} value={String(value ?? '')} onChange={onChange} />
    case 'products': return <ProductsPicker label={label} value={(value as string[]) ?? []} max={field.max} onChange={onChange} />
    case 'collections': return <CollectionsPicker label={label} value={(value as string[]) ?? []} max={field.max} onChange={onChange} />
    case 'links': return <LinksField label={label} value={(value as NavLink[]) ?? []} max={field.max} onChange={onChange} />
    case 'socialLinks': return <SocialLinksField label={label} value={(value as SocialLink[]) ?? []} max={field.max} onChange={onChange} />
    default: {
      const unknownKind: never = field
      console.warn('editor: no field for prop', name, unknownKind)
      return null
    }
  }
}

function NoSettings() {
  const { t } = useTranslation('editor')
  return <p className="text-xs text-neutral-500">{t('inspector.noSettings')}</p>
}

/** Editing one language other than the store's default: only texts, kept
 *  in the node's translation for that language. */
export interface Translating {
  translation: Props
  onChange: (next: Props) => void
}

/** The component's own settings, built from its definition. */
export function PropsForm({ def, props, onChange, translating }: { def: ComponentDefinition; props: Props; onChange: (next: Props) => void; translating?: Translating }) {
  const { t } = useTranslation('editor')
  const entries = Object.entries(def.props) as [string, PropField][]
  const shown = entries.filter(([name]) => def.showField?.[name]?.(props as never) ?? true)
  const visible = translating ? shown.filter(([, field]) => isTranslatable(field)) : shown
  if (!visible.length) return translating ? <p className="text-xs text-neutral-500">{t('inspector.nothingToTranslate')}</p> : <NoSettings />
  return (
    <div className="flex flex-col gap-3">
      {translating && <p className="text-xs text-neutral-500">{t('inspector.translatingNote')}</p>}
      {visible.map(([name, field]) => translating ? (
        <div key={name}>
          <FieldFor def={def} name={name} field={field} value={translating.translation[name] ?? props[name]}
            onChange={v => translating.onChange({ ...translating.translation, [name]: v })} />
          {!(name in translating.translation) && <p className="mt-1 text-2xs text-amber-700">{t('inspector.untranslated')}</p>}
        </div>
      ) : (
        <FieldFor key={name} def={def} name={name} field={field} value={props[name]} onChange={v => onChange({ ...props, [name]: v })} />
      ))}
    </div>
  )
}

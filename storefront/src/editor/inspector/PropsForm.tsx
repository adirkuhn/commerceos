import { useTranslation } from 'react-i18next'
import type { ComponentDefinition, PropField } from '../../components/types'
import { useEditorLabels } from '../i18n/useEditorLabels'
import type { RichText } from '../../core/richText'
import { LinkField, NumberField, SelectField, TextField, ToggleField } from './fields'
import type { NavLink } from '../../components/site/site'
import { LinksField } from './LinksField'
import { AssetPicker, CategorySelect, CollectionSelect, CollectionsPicker, ProductPicker, ProductsPicker } from './pickers'
import { RichTextField } from './RichTextField'

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

/** The component's own settings, built from its definition. */
export function PropsForm({ def, props, onChange }: { def: ComponentDefinition; props: Props; onChange: (next: Props) => void }) {
  const entries = Object.entries(def.props) as [string, PropField][]
  const visible = entries.filter(([name]) => def.showField?.[name]?.(props as never) ?? true)
  if (!visible.length) return <NoSettings />
  return (
    <div className="flex flex-col gap-3">
      {visible.map(([name, field]) => (
        <FieldFor key={name} def={def} name={name} field={field} value={props[name]} onChange={v => onChange({ ...props, [name]: v })} />
      ))}
    </div>
  )
}

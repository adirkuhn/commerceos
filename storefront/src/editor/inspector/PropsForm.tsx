import type { ComponentDefinition, PropField } from '../../components/types'
import type { RichText } from '../../core/richText'
import { LinkField, NumberField, SelectField, TextField, ToggleField } from './fields'
import type { NavLink } from '../../components/site/site'
import { LinksField } from './LinksField'
import { AssetPicker, CategorySelect, CollectionSelect, CollectionsPicker, ProductPicker, ProductsPicker } from './pickers'
import { RichTextField } from './RichTextField'

type Props = Record<string, unknown>

function FieldFor({ name, field, value, onChange }: { name: string; field: PropField; value: unknown; onChange: (v: unknown) => void }) {
  switch (field.kind) {
    case 'text': return <TextField label={field.label} value={String(value ?? '')} maxLength={field.maxLength} multiline={field.multiline} onChange={onChange} />
    case 'number': return <NumberField label={field.label} value={Number(value) || 0} min={field.min} max={field.max} onChange={onChange} />
    case 'select': return <SelectField label={field.label} value={(value ?? '') as string | number} options={field.options} onChange={onChange} />
    case 'toggle': return <ToggleField label={field.label} value={Boolean(value)} onChange={onChange} />
    case 'richtext': return <RichTextField label={field.label} value={(value as RichText) ?? []} onChange={onChange} />
    case 'asset': return <AssetPicker label={field.label} value={String(value ?? '')} onChange={onChange} />
    case 'link': return <LinkField label={field.label} value={String(value ?? '')} onChange={onChange} />
    case 'category': return <CategorySelect label={field.label} value={String(value ?? '')} onChange={onChange} />
    case 'collection': return <CollectionSelect label={field.label} value={String(value ?? '')} onChange={onChange} />
    case 'product': return <ProductPicker label={field.label} value={String(value ?? '')} onChange={onChange} />
    case 'products': return <ProductsPicker label={field.label} value={(value as string[]) ?? []} max={field.max} onChange={onChange} />
    case 'collections': return <CollectionsPicker label={field.label} value={(value as string[]) ?? []} max={field.max} onChange={onChange} />
    case 'links': return <LinksField label={field.label} value={(value as NavLink[]) ?? []} max={field.max} onChange={onChange} />
    case 'hidden': return null
    default: {
      const unknownKind: never = field
      console.warn('editor: no field for prop', name, unknownKind)
      return null
    }
  }
}

/** The component's own settings, built from its definition. */
export function PropsForm({ def, props, onChange }: { def: ComponentDefinition; props: Props; onChange: (next: Props) => void }) {
  const entries = Object.entries(def.props) as [string, PropField][]
  const visible = entries.filter(([name]) => def.showField?.[name]?.(props as never) ?? true)
  if (!visible.length) return <p className="text-xs text-neutral-500">This component has no settings.</p>
  return (
    <div className="flex flex-col gap-3">
      {visible.map(([name, field]) => (
        <FieldFor key={name} name={name} field={field} value={props[name]} onChange={v => onChange({ ...props, [name]: v })} />
      ))}
    </div>
  )
}

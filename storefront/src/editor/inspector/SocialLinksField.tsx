import { useTranslation } from 'react-i18next'
import { SOCIAL_NETWORKS, type SocialLink } from '../../components/site/social'
import { LinkField, SelectField } from './fields'
import { moveItem, removeItem, replaceItem } from './listOps'

const NETWORK_NAMES: Record<string, string> = {
  whatsapp: 'WhatsApp', instagram: 'Instagram', facebook: 'Facebook', tiktok: 'TikTok', youtube: 'YouTube', x: 'X',
}

/** An ordered list of social links: which network, and where it goes. */
export function SocialLinksField({ label, value, max, onChange }: { label: string; value: SocialLink[]; max: number; onChange: (v: SocialLink[]) => void }) {
  const { t } = useTranslation(['editor', 'shop'])
  const name = (n: string) => NETWORK_NAMES[n] ?? ({ maps: t('shop:social.maps'), email: t('shop:social.email'), phone: t('shop:social.phone') } as Record<string, string>)[n] ?? n
  const options = SOCIAL_NETWORKS.map(n => ({ value: n, label: name(n) }))
  return (
    <fieldset className="flex flex-col gap-2">
      <legend className="text-xs font-medium text-neutral-700">{label} <span className="text-neutral-500">({t('socialLinks.chosenOf', { chosen: value.length, max })})</span></legend>
      <ol className="flex flex-col gap-2" aria-label={label}>
        {value.map((item, i) => (
          <li key={i} className="flex flex-col gap-1 rounded border border-neutral-200 p-2">
            <SelectField label={t('socialLinks.network', { number: i + 1 })} value={item.network} options={options}
              onChange={v => v && onChange(replaceItem(value, i, { ...item, network: v }))} />
            <LinkField label={t('socialLinks.target', { number: i + 1 })} value={item.link} onChange={v => onChange(replaceItem(value, i, { ...item, link: v }))} />
            <div className="flex justify-end gap-2 text-sm">
              <button type="button" aria-label={t('links.moveUp', { number: i + 1 })} disabled={i === 0} onClick={() => onChange(moveItem(value, i, -1))}>↑</button>
              <button type="button" aria-label={t('links.moveDown', { number: i + 1 })} disabled={i === value.length - 1} onClick={() => onChange(moveItem(value, i, 1))}>↓</button>
              <button type="button" aria-label={t('links.remove', { number: i + 1 })} onClick={() => onChange(removeItem(value, i))}>✕</button>
            </div>
          </li>
        ))}
      </ol>
      {value.length < max && (
        <button type="button" className="rounded-md border px-2 py-1 text-sm"
          onClick={() => onChange([...value, { network: 'whatsapp', link: 'https://wa.me/' }])}>{t('socialLinks.add')}</button>
      )}
    </fieldset>
  )
}

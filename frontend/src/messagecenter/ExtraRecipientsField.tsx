import { useTranslation } from 'react-i18next'

interface Props {
  value: string
  onChange: (text: string) => void
}

// The rule form's "also email" list: addresses that always get this rule's
// email on top of whoever the trigger goes to. Kept as raw text while
// editing; the panel parses it with parseEmailList on save.
export function ExtraRecipientsField({ value, onChange }: Props) {
  const { t } = useTranslation('messages')
  return (
    <div>
      <label htmlFor="rule-extra-recipients" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
        {t('rules.extraRecipients')}
      </label>
      <textarea
        id="rule-extra-recipients"
        rows={2}
        value={value}
        onChange={e => onChange(e.target.value)}
        placeholder={t('rules.extraRecipientsPlaceholder')}
        className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus font-mono"
      />
      <p className="text-2xs text-ink-4 mt-1">{t('rules.extraRecipientsHint')}</p>
    </div>
  )
}

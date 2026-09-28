import { useEffect, useState } from 'react'
import type { RichText } from '../../core/richText'
import { TextField } from './fields'
import { plainToRichText, richTextToPlain } from './richTextPlain'

/** Edits rich text as plain text. The typed text is kept as-is (so a
 *  trailing newline survives while typing) and only replaced when the
 *  value changes from outside, e.g. undo or selecting another component. */
export function RichTextField({ label, value, onChange }: { label: string; value: RichText; onChange: (v: RichText) => void }) {
  const [draft, setDraft] = useState(() => richTextToPlain(value))
  useEffect(() => {
    setDraft(current => (richTextToPlain(plainToRichText(current)) === richTextToPlain(value) ? current : richTextToPlain(value)))
  }, [value])
  return (
    <TextField label={label} multiline value={draft} onChange={v => {
      setDraft(v)
      onChange(plainToRichText(v))
    }} />
  )
}

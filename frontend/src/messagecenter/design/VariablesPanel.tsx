import { useTranslation } from 'react-i18next'
import type { TemplateVariable } from '../types'
import { listSnippet, tokenFor } from './designHelpers'

interface Props {
  variables: TemplateVariable[]
  onInsertText: (token: string) => void
  onInsertHTML: (html: string) => void
}

function exampleText(v: TemplateVariable): string {
  return v.example === null || v.example === undefined || v.example === '' ? '' : String(v.example)
}

// VariablesPanel lists what the chosen trigger's templates can use. A click
// inserts a value at the cursor (or into the selected block); a list
// inserts a block that repeats once per entry.
export function VariablesPanel({ variables, onInsertText, onInsertHTML }: Props) {
  const { t } = useTranslation('messages')
  return (
    <div className="p-3 space-y-1">
      <h3 className="text-xs font-semibold text-ink mb-1">{t('design.variables')}</h3>
      <p className="text-2xs text-ink-4 mb-2">{t('design.variablesHint')}</p>
      {variables.map(v => (
        <div key={v.name}>
          <button
            type="button"
            onMouseDown={e => e.preventDefault()}
            onClick={() => (v.type === 'list' ? onInsertHTML(listSnippet(v)) : onInsertText(tokenFor(v)))}
            className="w-full text-left px-2 py-1.5 rounded-lg hover:bg-hover"
            title={t('design.insert')}
          >
            <span className="block font-mono text-2xs text-accent">{v.type === 'list' ? `{{#${v.name}}}` : tokenFor(v)}</span>
            {v.type === 'list'
              ? <span className="block text-2xs text-ink-4">{t('design.listOf', { fields: v.fields.map(f => f.name).join(', ') })}</span>
              : exampleText(v) && <span className="block text-2xs text-ink-4 truncate">{exampleText(v)}</span>}
          </button>
          {v.type === 'list' && (
            <div className="pl-4">
              {v.fields.map(f => (
                <button
                  key={f.name}
                  type="button"
                  onMouseDown={e => e.preventDefault()}
                  onClick={() => onInsertText(tokenFor(f))}
                  className="block w-full text-left px-2 py-1 rounded hover:bg-hover font-mono text-2xs text-ink-3"
                >
                  {tokenFor(f)}
                </button>
              ))}
            </div>
          )}
        </div>
      ))}
    </div>
  )
}

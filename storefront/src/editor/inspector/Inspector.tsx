import { useEffect, useReducer, useState, useSyncExternalStore } from 'react'
import { useTranslation } from 'react-i18next'
import type { Component, Editor } from 'grapesjs'
import { registry } from '../../components/registry'
import type { DocumentKind, NodeStyle, ThemeSettings } from '../../core/document'
import type { DeviceId } from '../useGrapesEditor'
import { PropsForm } from './PropsForm'
import { StyleForm } from './StyleForm'
import { SettingsPanel } from './SettingsPanel'
import { PageTextsForm, type PageTexts } from './PageTextsForm'
import { breakpointFor } from './styleControls'
import { useEditorLabels } from '../i18n/useEditorLabels'
import { useEditorServices } from '../services'
import { CopyIcon, ParentIcon, TrashIcon } from '../icons'

const actionBtn = 'rounded-ed p-1.5 text-ed-ink-3 transition-colors hover:bg-ed-hover hover:text-ed-ink'

type Translations = Record<string, Record<string, unknown>>
const translationsOf = (c: Component): Translations => (c.get('cosTranslations') ?? {}) as Translations

interface Props {
  editor: Editor
  device: DeviceId
  settings: ThemeSettings
  onSaveSettings: (s: ThemeSettings) => void
  savingSettings: boolean
  /** The open document's own texts, shown with the theme settings. */
  page: { kind: DocumentKind; texts: PageTexts; onChange: (t: PageTexts) => void }
}

/** Right-hand panel: the selected component's settings and styles, or the
 *  theme settings when nothing is selected. Every change goes through the
 *  component model, so GrapesJS undo/redo covers it. */
export function Inspector({ editor, device, settings, onSaveSettings, savingSettings, page }: Props) {
  const [selected, setSelected] = useState<Component | undefined>(editor.getSelected())
  const [, rerender] = useReducer((n: number) => n + 1, 0)
  const [tab, setTab] = useState<'content' | 'style'>('content')
  const { t } = useTranslation('editor')
  const labels = useEditorLabels()
  const services = useEditorServices()
  const contentLocale = useSyncExternalStore(services.contentLocale.subscribe, services.contentLocale.snapshot)

  useEffect(() => {
    const onSelect = () => setSelected(editor.getSelected())
    const onUpdate = (c: Component) => { if (c === editor.getSelected()) rerender() }
    editor.on('component:toggled', onSelect)
    editor.on('component:update', onUpdate)
    editor.on('undo redo', rerender)
    return () => {
      editor.off('component:toggled', onSelect)
      editor.off('component:update', onUpdate)
      editor.off('undo redo', rerender)
    }
  }, [editor])

  const def = selected ? registry.get(String(selected.get('type'))) : undefined
  if (!selected || !def) {
    return (
      <div className="flex flex-col gap-4">
        <PageTextsForm kind={page.kind} texts={page.texts} locale={contentLocale} onChange={page.onChange} />
        <SettingsPanel settings={settings} onSave={onSaveSettings} saving={savingSettings} />
      </div>
    )
  }

  const parent = selected.parent()
  const duplicate = () => {
    if (!parent) return
    parent.components().add(selected.clone(), { at: selected.index() + 1 })
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-2">
        <h2 className="font-semibold text-ed-ink">{labels.componentName(def)}</h2>
        <div className="ml-auto flex gap-0.5">
          {parent && parent.get('type') !== 'wrapper' && (
            <button type="button" className={actionBtn} onClick={() => editor.select(parent)} aria-label={t('inspector.selectParent')} title={t('inspector.selectParent')}><ParentIcon /></button>
          )}
          {def.editor.copyable && (
            <button type="button" className={actionBtn} onClick={duplicate} aria-label={t('inspector.duplicate')} title={t('inspector.duplicate')}><CopyIcon /></button>
          )}
          {def.editor.removable && (
            <button type="button" className={`${actionBtn} hover:!bg-ed-danger/10 hover:!text-ed-danger`} aria-label={t('inspector.delete')} title={t('inspector.delete')}
              onClick={() => { selected.remove(); editor.select(undefined) }}><TrashIcon /></button>
          )}
        </div>
      </div>
      <p className="text-xs text-ed-ink-4">{labels.componentDescription(def)}</p>
      <div role="tablist" className="flex gap-1 border-b border-ed-line">
        {(['content', 'style'] as const).map(name => (
          <button key={name} role="tab" aria-selected={tab === name} type="button" onClick={() => setTab(name)}
            className={`-mb-px px-3 py-1.5 text-sm ${tab === name ? 'border-b-2 border-ed-accent font-medium text-ed-ink' : 'text-ed-ink-3 hover:text-ed-ink'}`}>
            {name === 'content' ? t('inspector.content') : t('inspector.style')}
          </button>
        ))}
      </div>
      {tab === 'content' ? (
        <PropsForm def={def} props={selected.get('cosProps') ?? {}} onChange={next => selected.set('cosProps', next)}
          translating={contentLocale ? {
            translation: translationsOf(selected)[contentLocale] ?? {},
            onChange: next => selected.set('cosTranslations', { ...translationsOf(selected), [contentLocale]: next }),
          } : undefined} />
      ) : (
        <StyleForm groups={def.styleGroups} style={(selected.get('cosStyle') ?? {}) as NodeStyle} breakpoint={breakpointFor(device)}
          onChange={next => selected.set('cosStyle', next)} />
      )}
    </div>
  )
}

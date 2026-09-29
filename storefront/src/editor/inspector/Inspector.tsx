import { useEffect, useReducer, useState, useSyncExternalStore } from 'react'
import { useTranslation } from 'react-i18next'
import type { Component, Editor } from 'grapesjs'
import { registry } from '../../components/registry'
import type { NodeStyle, ThemeSettings } from '../../core/document'
import type { DeviceId } from '../useGrapesEditor'
import { PropsForm } from './PropsForm'
import { StyleForm } from './StyleForm'
import { SettingsPanel } from './SettingsPanel'
import { breakpointFor } from './styleControls'
import { useEditorLabels } from '../i18n/useEditorLabels'
import { useEditorServices } from '../services'

type Translations = Record<string, Record<string, unknown>>
const translationsOf = (c: Component): Translations => (c.get('cosTranslations') ?? {}) as Translations

interface Props {
  editor: Editor
  device: DeviceId
  settings: ThemeSettings
  onSaveSettings: (s: ThemeSettings) => void
  savingSettings: boolean
}

/** Right-hand panel: the selected component's settings and styles, or the
 *  theme settings when nothing is selected. Every change goes through the
 *  component model, so GrapesJS undo/redo covers it. */
export function Inspector({ editor, device, settings, onSaveSettings, savingSettings }: Props) {
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
    return <SettingsPanel settings={settings} onSave={onSaveSettings} saving={savingSettings} />
  }

  const parent = selected.parent()
  const duplicate = () => {
    if (!parent) return
    parent.components().add(selected.clone(), { at: selected.index() + 1 })
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-2">
        <h2 className="font-semibold">{labels.componentName(def)}</h2>
        <div className="ml-auto flex gap-1 text-xs">
          {parent && parent.get('type') !== 'wrapper' && <button type="button" onClick={() => editor.select(parent)}>{t('inspector.selectParent')}</button>}
          {def.editor.copyable && <button type="button" onClick={duplicate}>{t('inspector.duplicate')}</button>}
          {def.editor.removable && <button type="button" className="text-red-700" onClick={() => { selected.remove(); editor.select(undefined) }}>{t('inspector.delete')}</button>}
        </div>
      </div>
      <p className="text-xs text-neutral-500">{labels.componentDescription(def)}</p>
      <div role="tablist" className="flex gap-1 border-b">
        {(['content', 'style'] as const).map(name => (
          <button key={name} role="tab" aria-selected={tab === name} type="button" onClick={() => setTab(name)}
            className={`px-3 py-1.5 text-sm ${tab === name ? 'border-b-2 border-blue-700 font-medium' : 'text-neutral-600'}`}>
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

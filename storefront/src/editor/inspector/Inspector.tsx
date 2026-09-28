import { useEffect, useReducer, useState } from 'react'
import type { Component, Editor } from 'grapesjs'
import { registry } from '../../components/registry'
import type { NodeStyle, ThemeSettings } from '../../core/document'
import type { DeviceId } from '../useGrapesEditor'
import { PropsForm } from './PropsForm'
import { StyleForm } from './StyleForm'
import { SettingsPanel } from './SettingsPanel'
import { breakpointFor } from './styleControls'

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
        <h2 className="font-semibold">{def.name}</h2>
        <div className="ml-auto flex gap-1 text-xs">
          {parent && parent.get('type') !== 'wrapper' && <button type="button" onClick={() => editor.select(parent)}>Select parent</button>}
          {def.editor.copyable && <button type="button" onClick={duplicate}>Duplicate</button>}
          {def.editor.removable && <button type="button" className="text-red-700" onClick={() => { selected.remove(); editor.select(undefined) }}>Delete</button>}
        </div>
      </div>
      <p className="text-xs text-neutral-500">{def.description}</p>
      <div role="tablist" className="flex gap-1 border-b">
        {(['content', 'style'] as const).map(t => (
          <button key={t} role="tab" aria-selected={tab === t} type="button" onClick={() => setTab(t)}
            className={`px-3 py-1.5 text-sm ${tab === t ? 'border-b-2 border-blue-700 font-medium' : 'text-neutral-600'}`}>
            {t === 'content' ? 'Content' : 'Style'}
          </button>
        ))}
      </div>
      {tab === 'content' ? (
        <PropsForm def={def} props={selected.get('cosProps') ?? {}} onChange={next => selected.set('cosProps', next)} />
      ) : (
        <StyleForm groups={def.styleGroups} style={(selected.get('cosStyle') ?? {}) as NodeStyle} breakpoint={breakpointFor(device)}
          onChange={next => selected.set('cosStyle', next)} />
      )}
    </div>
  )
}

// @vitest-environment jsdom
import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

// GrapesJS needs a real browser: the test checks what the hook asks of it.
const fake = vi.hoisted(() => {
  const handlers: Record<string, Array<(...args: unknown[]) => void>> = {}
  const editor = {
    BlockManager: { add: vi.fn() },
    Panels: { getPanels: vi.fn(() => ({ reset: vi.fn() })) },
    UndoManager: { undo: vi.fn(), redo: vi.fn() },
    setDevice: vi.fn(),
    runCommand: vi.fn(),
    destroy: vi.fn(),
    getSelected: vi.fn(),
    on: vi.fn((event: string, fn: (...args: unknown[]) => void) => { (handlers[event] ??= []).push(fn) }),
  }
  return { editor, handlers, init: vi.fn(() => editor) }
})
vi.mock('grapesjs', () => ({ default: { init: fake.init } }))
vi.mock('grapesjs-preset-newsletter', () => ({ default: 'preset-newsletter' }))
vi.mock('grapesjs/locale/nl.mjs', () => ({ default: {} }))
vi.mock('grapesjs/locale/pt.mjs', () => ({ default: {} }))

import { useEmailEditor, type EmailEditorOptions } from '../useEmailEditor'

function emit(event: string, ...args: unknown[]) {
  for (const fn of fake.handlers[event] ?? []) (fn as (...a: unknown[]) => void)(...args)
}

function mount(extra: Partial<EmailEditorOptions> = {}) {
  const el = (id: string) => Object.assign(document.createElement('div'), { id })
  const refs = { canvas: el('canvas'), blocks: el('blocks'), styles: el('styles'), layers: el('layers') }
  const opts: EmailEditorOptions = {
    html: '<p>Hi</p>',
    blocks: [{ id: 'cos-header', label: 'Store header', media: '<svg></svg>', content: '<table></table>' }],
    blockCategory: 'CommerceOS',
    locale: 'en',
    panels: { blocks: { current: refs.blocks }, styles: { current: refs.styles }, layers: { current: refs.layers } },
    sectorLabels: { text: 'Text', spacing: 'Spacing', size: 'Size', background: 'Background', border: 'Border' },
    uploadImage: vi.fn(),
    onUploadError: vi.fn(),
    ...extra,
  }
  const container = { current: refs.canvas }
  const hook = renderHook(() => useEmailEditor(container, opts))
  return { hook, refs }
}

interface InitConfig {
  blockManager: { appendTo: HTMLElement }
  styleManager: { appendTo: HTMLElement; sectors: Array<{ name: string; properties: Array<string | { property: string }> }> }
  layerManager: { appendTo: HTMLElement }
  pluginsOpts: Record<string, Record<string, unknown>>
}

function initConfig(): InitConfig {
  return (fake.init.mock.calls[0] as unknown as [InitConfig])[0]
}

beforeEach(() => {
  vi.clearAllMocks()
  for (const k of Object.keys(fake.handlers)) delete fake.handlers[k]
})

describe('useEmailEditor', () => {
  it('renders blocks, styles and layers into our own panels', () => {
    const { refs } = mount()
    const cfg = initConfig()

    expect(cfg.blockManager.appendTo).toBe(refs.blocks)
    expect(cfg.styleManager.appendTo).toBe(refs.styles)
    expect(cfg.layerManager.appendTo).toBe(refs.layers)
  })

  it('keeps the newsletter preset from drawing its own panels, theme and style list', () => {
    mount()
    const opts = initConfig().pluginsOpts['preset-newsletter']

    expect(opts).toMatchObject({ inlineCss: true, useCustomTheme: false, updateStyleManager: false, showBlocksOnLoad: false, showStylesOnChange: false })
  })

  it('removes the GrapesJS panel bar once loaded', () => {
    mount()
    const reset = vi.fn()
    fake.editor.Panels.getPanels.mockReturnValue({ reset })

    act(() => emit('load'))

    expect(reset).toHaveBeenCalledWith([])
  })

  it('offers only the style settings an email needs, under translated names', () => {
    mount()
    const sectors = initConfig().styleManager.sectors

    expect(sectors.map(s => s.name)).toEqual(['Text', 'Spacing', 'Size', 'Background', 'Border'])
    const props = sectors.flatMap(s => s.properties.map(p => (typeof p === 'string' ? p : p.property)))
    expect(props).toContain('font-family')
    expect(props).not.toContain('text-shadow')
    expect(props).not.toContain('vertical-align')
  })

  it('shows the email as a page in both device sizes', () => {
    mount()
    const devices = (initConfig() as unknown as { deviceManager: { devices: Array<{ name: string; width: string }> } }).deviceManager.devices

    expect(devices).toEqual([
      expect.objectContaining({ name: 'Desktop', width: '760px' }),
      expect.objectContaining({ name: 'Mobile portrait', width: '375px' }),
    ])
  })

  it('adds the CommerceOS blocks with their names and icons', () => {
    mount()

    expect(fake.editor.BlockManager.add).toHaveBeenCalledWith('cos-header', {
      label: 'Store header', media: '<svg></svg>', category: 'CommerceOS', content: '<table></table>',
    })
  })

  it('undoes, redoes, switches device and opens the code', () => {
    const { hook } = mount()

    hook.result.current.undo()
    hook.result.current.redo()
    act(() => hook.result.current.setDevice('mobile'))
    hook.result.current.showCode()

    expect(fake.editor.UndoManager.undo).toHaveBeenCalled()
    expect(fake.editor.UndoManager.redo).toHaveBeenCalled()
    expect(fake.editor.setDevice).toHaveBeenCalledWith('Mobile portrait')
    expect(hook.result.current.device).toBe('mobile')
    expect(fake.editor.runCommand).toHaveBeenCalledWith('export-template')
  })

  it('tells the page when a block is selected', () => {
    const onSelect = vi.fn()
    mount({ onSelect })

    act(() => emit('component:selected'))

    expect(onSelect).toHaveBeenCalled()
  })
})

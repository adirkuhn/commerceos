// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useState } from 'react'
import { registry } from '../../../components/registry'
import { defaultSettings } from '../../../core/settings'
import { StorefrontAPI } from '../../../api/storefront'
import { EditorAPI, type Asset } from '../../api'
import { AssetStore } from '../../assetStore'
import { EditorServicesProvider, type EditorServices } from '../../services'
import { PropsForm } from '../PropsForm'
import { StyleForm } from '../StyleForm'
import { SettingsPanel } from '../SettingsPanel'
import { AssetLibrary } from '../AssetLibrary'
import { LinkField } from '../fields'
import type { NodeStyle } from '../../../core/document'

function services(overrides: Partial<EditorServices> = {}): EditorServices {
  const store = new StorefrontAPI({ baseURL: 'x', storeKey: 'k' })
  store.listCategories = vi.fn(async () => [{ id: 'c1', parent_id: null, name: 'Shoes', slug: 'shoes', position: 0 }])
  store.listProducts = vi.fn(async () => ({ items: [], total: 0 }))
  return { api: new EditorAPI({ baseURL: 'x', token: 't' }), store, assets: new AssetStore(), queryClient: new QueryClient(), channelId: 'chan', ...overrides }
}

const wrap = (s: EditorServices) => ({ children }: { children: ReactNode }) => <EditorServicesProvider value={s}>{children}</EditorServicesProvider>

function Stateful({ type }: { type: string }) {
  const def = registry.get(type)!
  const [props, setProps] = useState<Record<string, unknown>>({ ...def.defaults })
  return <><PropsForm def={def} props={props} onChange={setProps} /><output data-testid="props">{JSON.stringify(props)}</output></>
}

describe('PropsForm', () => {
  it('shows only the fields that apply and edits props immutably', async () => {
    render(<Stateful type="product-grid" />, { wrapper: wrap(services()) })
    expect(screen.queryByLabelText('Category')).toBeNull()
    expect(screen.getByLabelText('How many')).toBeInTheDocument()

    await userEvent.selectOptions(screen.getByLabelText('Show'), 'category')
    await userEvent.selectOptions(await screen.findByLabelText('Category'), await screen.findByRole('option', { name: 'Shoes' }))

    expect(JSON.parse(screen.getByTestId('props').textContent!)).toMatchObject({ source: 'category', category_id: 'c1' })
  })

  it('edits rich text through the plain text box', async () => {
    render(<Stateful type="text" />, { wrapper: wrap(services()) })
    const box = screen.getByLabelText('Text')
    await userEvent.clear(box)
    await userEvent.type(box, '- one{enter}- two')
    expect(JSON.parse(screen.getByTestId('props').textContent!).content).toEqual([
      { type: 'bulleted_list', items: [[{ text: 'one' }], [{ text: 'two' }]] },
    ])
  })
})

describe('LinkField', () => {
  it('only commits valid links', async () => {
    const onChange = vi.fn()
    render(<LinkField label="Link" value="" onChange={onChange} />)
    await userEvent.type(screen.getByLabelText('Link'), 'javascript:x')
    expect(onChange).not.toHaveBeenCalledWith('javascript:x')
    expect(screen.getByLabelText('Link')).toHaveAttribute('aria-invalid', 'true')
  })
})

describe('StyleForm', () => {
  it('edits the current breakpoint and shows what it inherits', async () => {
    const onChange = vi.fn()
    const style: NodeStyle = { base: { padding_y: 'xl' } }
    render(<StyleForm groups={['spacing']} style={style} breakpoint="mobile" onChange={onChange} />, { wrapper: wrap(services()) })

    const select = screen.getByLabelText('Padding top/bottom')
    expect(within(select).getByRole('option', { name: /Same as larger screens \(xl\)/ })).toBeInTheDocument()
    await userEvent.selectOptions(select, 'sm')

    expect(onChange).toHaveBeenCalledWith({ base: { padding_y: 'xl' }, mobile: { padding_y: 'sm' } })
  })
})

describe('SettingsPanel', () => {
  it('saves changed settings only', async () => {
    const onSave = vi.fn()
    render(<SettingsPanel settings={defaultSettings()} onSave={onSave} saving={false} />, { wrapper: wrap(services()) })
    const save = screen.getByRole('button', { name: 'Save settings' })
    expect(save).toBeDisabled()

    await userEvent.selectOptions(screen.getByLabelText('Products per row'), '3')
    await userEvent.click(save)

    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ product: expect.objectContaining({ grid_columns: 3 }) }))
  })
})

describe('AssetLibrary', () => {
  const asset: Asset = { id: 'a1', url: 'https://cdn/a.png', filename: 'a.png', content_type: 'image/png', bytes: 1, width: 2, height: 2, alt: 'Logo', created_at: '' }

  it('uploads and picks images, remembering them for the canvas', async () => {
    const s = services()
    s.api.listAssets = vi.fn(async () => ({ items: [], total: 0 }))
    s.api.uploadAsset = vi.fn(async () => asset)
    const onPick = vi.fn()
    render(<AssetLibrary onPick={onPick} onClose={vi.fn()} />, { wrapper: wrap(s) })
    await screen.findByText(/No images yet/)

    await userEvent.upload(screen.getByLabelText('Upload image'), new File(['x'], 'a.png', { type: 'image/png' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Use Logo' }))

    expect(onPick).toHaveBeenCalledWith(asset)
    expect(s.assets.snapshot()).toEqual({ a1: { url: 'https://cdn/a.png', width: 2, height: 2, alt: 'Logo' } })
  })

  it('shows upload errors', async () => {
    const s = services()
    s.api.listAssets = vi.fn(async () => ({ items: [], total: 0 }))
    s.api.uploadAsset = vi.fn(async () => { throw new Error('only JPEG, PNG, WebP, GIF and ICO images can be uploaded') })
    render(<AssetLibrary onPick={vi.fn()} onClose={vi.fn()} />, { wrapper: wrap(s) })
    await userEvent.upload(await screen.findByLabelText('Upload image'), new File(['x'], 'a.png', { type: 'image/png' }))
    await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent('only JPEG'))
  })
})

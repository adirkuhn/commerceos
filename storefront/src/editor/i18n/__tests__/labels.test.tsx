// @vitest-environment jsdom
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { I18nextProvider } from 'react-i18next'
import { useState } from 'react'
import { QueryClient } from '@tanstack/react-query'
import { createEditorI18n as createI18n } from '../createEditorI18n'
import { registry } from '../../../components/registry'
import { defaultSettings } from '../../../core/settings'
import { StorefrontAPI } from '../../../api/storefront'
import { EditorAPI } from '../../api'
import { AssetStore } from '../../assetStore'
import { PreviewStore } from '../../previewStore'
import { ValueStore } from '../../valueStore'
import { EditorServicesProvider, type EditorServices } from '../../services'
import { PropsForm } from '../../inspector/PropsForm'
import { StyleForm } from '../../inspector/StyleForm'
import { SettingsPanel } from '../../inspector/SettingsPanel'
import { createEditorLabels } from '../labels'
import { editorLocale } from '../../editorLocale'

function services(): EditorServices {
  const store = new StorefrontAPI({ baseURL: 'x', storeKey: 'k' })
  store.listCategories = async () => []
  store.listCollections = async () => []
  store.listProducts = async () => ({ items: [], total: 0 })
  return {
    api: new EditorAPI({ baseURL: 'x', token: 't' }), store, assets: new AssetStore(), preview: new PreviewStore(),
    settings: new ValueStore(defaultSettings()), queryClient: new QueryClient(), channelId: 'chan', contentLocale: new ValueStore<'' | 'en' | 'pt-BR' | 'nl'>(''),
  }
}

function Hero() {
  const def = registry.get('hero')!
  const [props, setProps] = useState<Record<string, unknown>>({ ...def.defaults })
  return <PropsForm def={def} props={props} onChange={setProps} />
}

const inPortuguese = (el: React.ReactNode) =>
  render(<I18nextProvider i18n={createI18n('pt-BR')}><EditorServicesProvider value={services()}>{el}</EditorServicesProvider></I18nextProvider>)

describe('editor in the admin user’s language', () => {
  it('shows component settings, options and styles in Portuguese', async () => {
    inPortuguese(<><Hero /><StyleForm groups={['spacing', 'typography']} style={{}} breakpoint="base" onChange={() => undefined} /></>)
    expect(screen.getByLabelText('Título')).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'Centralizado' })).toBeInTheDocument()
    expect(screen.getByText('Espaçamento')).toBeInTheDocument()
    expect(screen.getByLabelText('Espaço acima')).toBeInTheDocument()
    // Size tokens read the same in every language.
    expect(screen.getAllByRole('option', { name: '2xl' }).length).toBeGreaterThan(0)
    expect(screen.getByText(/Editando o estilo para todas as telas/)).toBeInTheDocument()
  })

  it('shows theme settings in Portuguese', async () => {
    inPortuguese(<SettingsPanel settings={defaultSettings()} onSave={() => undefined} saving={false} />)
    expect(screen.getByRole('heading', { name: 'Configurações do tema' })).toBeInTheDocument()
    expect(screen.getByLabelText('Cor Primária')).toBeInTheDocument()
    await userEvent.selectOptions(screen.getByLabelText('Cantos dos botões'), 'Pílula')
    expect(screen.getByRole('button', { name: 'Salvar configurações' })).toBeEnabled()
  })
})

describe('createEditorLabels', () => {
  it('falls back to the English in the code and keeps tokens', () => {
    const t = createI18n('nl').getFixedT(null, 'editor')
    const labels = createEditorLabels(t)
    const grid = registry.get('product-grid')!
    expect(labels.componentName(grid)).toBe('Productraster')
    expect(labels.category('commerce')).toBe('Winkel')
    expect(labels.option(grid, 'columns', { value: 3, label: '3' })).toBe('3')
    expect(labels.field({ ...grid, type: 'brand-new' }, 'limit', { kind: 'number', label: 'How many' })).toBe('How many')
    expect(labels.document({ kind: 'template', key: 'order-confirmation', title: 'Order confirmation' })).toBe('Orderbevestiging')
    expect(labels.document({ kind: 'page', key: 'about-us', title: '' })).toBe('About Us')
  })
})

describe('editorLocale', () => {
  it('uses the session language, else the browser’s', () => {
    expect(editorLocale('pt-BR')).toBe('pt-BR')
    expect(editorLocale('nl-NL')).toBe('nl')
    expect(editorLocale(undefined)).toBe(editorLocale(navigator.language))
  })
})

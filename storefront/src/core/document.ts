// Theme document types -- hand-written mirror of backend/api/theme/openapi.yaml
// (ThemeNode, Document, ThemeSettings...). Contract tests keep them honest.

export type DocumentKind = 'template' | 'page' | 'section_group'

export type StyleValues = Partial<Record<StyleProperty, string>>

export type StyleProperty =
  | 'padding_x' | 'padding_y' | 'margin_top' | 'margin_bottom' | 'gap'
  | 'text_align' | 'align_items' | 'justify_content'
  | 'font_family' | 'font_size' | 'font_weight' | 'line_height' | 'letter_spacing' | 'text_transform'
  | 'color' | 'background_color' | 'border_color' | 'border_width' | 'border_radius' | 'shadow'
  | 'width' | 'min_height' | 'aspect_ratio'
  | 'background_image' | 'background_size' | 'background_position'
  | 'columns' | 'display'

export interface NodeStyle {
  base?: StyleValues
  tablet?: StyleValues
  mobile?: StyleValues
}

export type Breakpoint = keyof NodeStyle

export interface ThemeNode {
  id: string
  type: string
  props?: Record<string, unknown>
  /** Per-language overrides of translatable props; props are the store's
   *  default language (see components/translatable.ts). */
  translations?: Record<string, Record<string, unknown>>
  style?: NodeStyle
  children?: ThemeNode[]
}

export interface SEO {
  title?: string
  description?: string
}

export interface AssetRef {
  url: string
  width: number
  height: number
  alt: string
}

/** A document's merchant texts in one language. */
export interface DocumentTexts {
  /** Pages only. */
  title?: string
  seo?: SEO
}

export interface ThemeDocument {
  kind: DocumentKind
  key: string
  title: string
  seo: SEO
  /** Title/SEO in the store's other languages, keyed by language. */
  translations?: Record<string, DocumentTexts>
  root: ThemeNode
  /** Images the document refers to ("asset:<id>"), keyed by id. */
  assets?: Record<string, AssetRef>
}

export interface DocumentSummary {
  kind: DocumentKind
  key: string
  title: string
}

export type SpacingScale = 'none' | 'xs' | 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '3xl'

export interface ThemeSettings {
  brand: { logo_asset_id?: string | null; favicon_asset_id?: string | null }
  colors: { primary: string; secondary: string; background: string; surface: string; text: string; muted: string; accent: string }
  typography: { source: 'system' | 'google'; heading_font: string; body_font: string; scale: 'sm' | 'md' | 'lg' }
  layout: { max_width: number; gutter: SpacingScale }
  buttons: { radius: 'none' | 'sm' | 'md' | 'lg' | 'full'; size: 'sm' | 'md' | 'lg'; style: 'solid' | 'outline' }
  product: { grid_columns: number; image_ratio: '1:1' | '4:3' | '3:4' | '2:3' | '16:9' }
  header: { layout: 'logo-left' | 'logo-center'; sticky: boolean }
  footer: { layout: 'columns' | 'simple' }
}

export interface StorefrontTheme {
  version_id: string
  version_number: number
  preview: boolean
  settings: ThemeSettings
  documents: DocumentSummary[]
  assets?: Record<string, AssetRef>
}

export const ROOT_TYPE = 'root'

/** The document as read in one language: its translated title/SEO over the
 *  default-language ones (blank = the default). */
export function documentIn<D extends ThemeDocument>(doc: D, locale: string | undefined): D {
  const tr = locale ? doc.translations?.[locale] : undefined
  if (!tr) return doc
  return {
    ...doc,
    title: tr.title || doc.title,
    seo: { title: tr.seo?.title || doc.seo?.title, description: tr.seo?.description || doc.seo?.description },
  }
}

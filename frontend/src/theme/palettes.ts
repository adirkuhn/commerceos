// A palette maps onto storefront-frontend's Tailwind `brand` color scale
// (see storefront-frontend/tailwind.config.js), which every block component
// uses for text/borders/buttons -- applying one just means setting these
// five CSS custom properties via the page's own custom_css field
// (storefront-frontend's CanvasPage renders custom_css into a <style> tag).
export interface ThemePalette {
  id: string
  label: string
  // 500 is what a swatch preview shows -- the scale's midpoint, used for
  // primary actions/links throughout the storefront.
  colors: { 50: string; 100: string; 500: string; 600: string; 700: string }
}

export const THEME_PALETTES: ThemePalette[] = [
  {
    id: 'slate',
    label: 'Carvão (default)',
    colors: { 50: '#f5f6f7', 100: '#e9ebee', 500: '#2b2f36', 600: '#1c1f24', 700: '#111318' },
  },
  {
    id: 'verde',
    label: 'Brasil Verde',
    colors: { 50: '#f0faf1', 100: '#dbf2de', 500: '#1f8a3d', 600: '#166b2f', 700: '#0f4d21' },
  },
  {
    id: 'amazonia',
    label: 'Amazônia',
    colors: { 50: '#fff8ec', 100: '#ffedc7', 500: '#d97706', 600: '#b45309', 700: '#92400e' },
  },
  {
    id: 'oceano',
    label: 'Oceano',
    colors: { 50: '#eff8ff', 100: '#d9edff', 500: '#0369a1', 600: '#075985', 700: '#0c4a6e' },
  },
  {
    id: 'vinho',
    label: 'Vinho',
    colors: { 50: '#fdf2f4', 100: '#fbdce2', 500: '#9d174d', 600: '#831843', 700: '#500724' },
  },
  // Derived from a merchant-supplied Brazilian grocery storefront design
  // (deep forest green on a warm cream background) -- see the Mercado
  // starter template in starterTemplate.ts, which pairs naturally with
  // this palette but isn't tied to it; either can be picked independently.
  {
    id: 'canto',
    label: 'Verde Cerrado',
    colors: { 50: '#fbf9f4', 100: '#f1eadf', 500: '#1b4b31', 600: '#143925', 700: '#0d2619' },
  },
]

export function paletteCustomCSS(palette: ThemePalette): string {
  const c = palette.colors
  return `:root {\n  --brand-50: ${c[50]};\n  --brand-100: ${c[100]};\n  --brand-500: ${c[500]};\n  --brand-600: ${c[600]};\n  --brand-700: ${c[700]};\n}`
}

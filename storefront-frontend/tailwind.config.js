/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        // CSS custom properties, not fixed hex -- a merchant's chosen
        // theme palette (see frontend/src/theme/palettes.ts) sets these
        // via a ThemePage's custom_css (rendered into a <style> tag by
        // CanvasPage), so the palette applies at runtime without
        // rebuilding this app. The literal fallback is what renders with
        // no custom_css at all -- today's original default palette.
        brand: {
          50: 'var(--brand-50, #f5f6f7)',
          100: 'var(--brand-100, #e9ebee)',
          500: 'var(--brand-500, #2b2f36)',
          600: 'var(--brand-600, #1c1f24)',
          700: 'var(--brand-700, #111318)',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
      },
    },
  },
  plugins: [],
}

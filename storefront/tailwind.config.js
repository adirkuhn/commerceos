/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './editor.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      // The theme editor's chrome tokens (src/theme/editor.css); the
      // shopper storefront never uses them.
      colors: Object.fromEntries([
        'canvas', 'surface', 'raised', 'sunken', 'fill', 'hover', 'line', 'ink', 'ink-2', 'ink-3', 'ink-4',
        'accent', 'accent-solid', 'accent-solid-hover', 'danger', 'success', 'warning',
      ].map(k => [`ed-${k}`, `rgb(var(--ed-${k}) / <alpha-value>)`])),
      borderRadius: { ed: 'var(--ed-radius)' },
    },
  },
  plugins: [],
}

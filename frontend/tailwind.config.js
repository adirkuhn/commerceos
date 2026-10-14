import plugin from 'tailwindcss/plugin'

// Padding and gaps follow the shop's spacing choice (--density, set by the
// appearance setting on the depot design only). Unset it is 1, so the
// computed sizes -- and classic -- stay exactly Tailwind's.
function densityScaled(spacing) {
  return Object.fromEntries(Object.entries(spacing).map(([k, v]) => [
    k, typeof v === 'string' && v.endsWith('rem') ? `calc(${v} * var(--density, 1))` : v,
  ]))
}

/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    padding: ({ theme }) => densityScaled(theme('spacing')),
    gap: ({ theme }) => densityScaled(theme('spacing')),
    space: ({ theme }) => densityScaled(theme('spacing')),
    extend: {
      colors: {
        // Semantic tokens -- values live in src/index.css and switch with
        // the appearance. Prefer these over raw hex or gray-N.
        canvas: 'rgb(var(--canvas) / <alpha-value>)',
        surface: 'rgb(var(--surface) / <alpha-value>)',
        raised: 'rgb(var(--raised) / <alpha-value>)',
        sunken: 'rgb(var(--sunken) / <alpha-value>)',
        fill: 'rgb(var(--fill) / <alpha-value>)',
        hover: 'rgb(var(--hover) / <alpha-value>)',
        line: 'rgb(var(--line) / <alpha-value>)',
        ink: {
          DEFAULT: 'rgb(var(--ink) / <alpha-value>)',
          2: 'rgb(var(--ink-2) / <alpha-value>)',
          3: 'rgb(var(--ink-3) / <alpha-value>)',
          4: 'rgb(var(--ink-4) / <alpha-value>)',
        },
        // Brand accent: "postmark ultramarine" -- the ink of parcel stamps
        // and carrier labels. accent = text/icons/tints, accent-solid =
        // filled primary buttons (white text).
        accent: {
          DEFAULT: 'rgb(var(--accent) / <alpha-value>)',
          solid: 'rgb(var(--accent-solid) / <alpha-value>)',
          'solid-hover': 'rgb(var(--accent-solid-hover) / <alpha-value>)',
        },
        danger: 'rgb(var(--danger) / <alpha-value>)',
        success: 'rgb(var(--success) / <alpha-value>)',
        warning: 'rgb(var(--warning) / <alpha-value>)',
        focus: 'rgb(var(--focus) / <alpha-value>)',
        // The depot sidebar (depot design only; classic resolves to surface).
        depot: {
          DEFAULT: 'rgb(var(--depot) / <alpha-value>)',
          ink: 'rgb(var(--depot-ink) / <alpha-value>)',
          'ink-2': 'rgb(var(--depot-ink-2) / <alpha-value>)',
          hover: 'rgb(var(--depot-hover) / <alpha-value>)',
          mark: 'rgb(var(--depot-mark) / <alpha-value>)',
        },
        // Fixed dark palette for surfaces that are dark in both
        // appearances (the handheld scanner app) or one-off dark-only shades.
        night: {
          950: '#0B1220',
          900: '#111827',
          800: '#1F2937',
          700: '#2D3748',
        },
      },
      fontSize: {
        // Smallest text in the app (captions, badges, table meta). 11px is
        // the floor -- nothing renders smaller (HIG minimums: 10pt macOS,
        // 11pt iOS).
        '2xs': ['0.6875rem', { lineHeight: '1rem' }],
        // Desktop body size (macOS default 13pt) -- navigation and dense UI.
        body: ['0.8125rem', { lineHeight: '1.25rem' }],
      },
      fontFamily: {
        // Variables so the design (classic Inter / depot Atkinson Hyperlegible) and the shop's appearance can switch
        // at runtime -- values in src/index.css.
        sans: ['var(--font-sans)'],
        data: ['var(--font-data)'],
        mono: ['var(--font-mono)'],
      },
      borderRadius: {
        // Classic keeps Tailwind's defaults; depot uses two radii (6px
        // controls, 10px panels) -- values in src/index.css.
        md: 'var(--radius-md)',
        lg: 'var(--radius-lg)',
        xl: 'var(--radius-xl)',
        '2xl': 'var(--radius-2xl)',
      },
      keyframes: {
        'scan-line': {
          '0%, 100%': { top: '4%' },
          '50%': { top: '92%' },
        },
        // A packing line a scan just touched: a short pop.
        'pack-flash': {
          '0%': { transform: 'scale(1)' },
          '30%': { transform: 'scale(1.025)' },
          '100%': { transform: 'scale(1)' },
        },
      },
      animation: {
        'scan-line': 'scan-line 1.8s ease-in-out infinite',
        'pack-flash': 'pack-flash 0.45s ease-out',
      },
    },
  },
  plugins: [
    // depot:x applies x only in the depot design (html.design-depot), so a
    // screen can be restyled for depot while classic stays exactly as it
    // was -- e.g. "depot:flex-row depot:border-0".
    plugin(({ addVariant }) => { addVariant('depot', '.design-depot &') }),
  ],
}

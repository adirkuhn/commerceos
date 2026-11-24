// Padding and gaps follow the shop's spacing choice (--density, set by the
// appearance setting). Unset it is 1, so the computed sizes stay exactly
// Tailwind's.
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
        // The look's action colour (carrier navy in Ticket Wallet, console
        // blue in Graphite). accent = text/icons/tints, accent-solid =
        // filled primary buttons (white text).
        accent: {
          DEFAULT: 'rgb(var(--accent) / <alpha-value>)',
          solid: 'rgb(var(--accent-solid) / <alpha-value>)',
          'solid-hover': 'rgb(var(--accent-solid-hover) / <alpha-value>)',
        },
        // Filled figures: quantities, counts and totals (carbon purple in
        // Ticket Wallet, plain ink in Graphite).
        figure: 'rgb(var(--figure) / <alpha-value>)',
        danger: 'rgb(var(--danger) / <alpha-value>)',
        success: 'rgb(var(--success) / <alpha-value>)',
        warning: 'rgb(var(--warning) / <alpha-value>)',
        focus: 'rgb(var(--focus) / <alpha-value>)',
        // The sidebar rail: the look's own colour in both appearances.
        rail: {
          DEFAULT: 'rgb(var(--rail) / <alpha-value>)',
          active: 'rgb(var(--rail-active) / <alpha-value>)',
          ink: 'rgb(var(--rail-ink) / <alpha-value>)',
          'ink-2': 'rgb(var(--rail-ink-2) / <alpha-value>)',
          hover: 'rgb(var(--rail-hover) / <alpha-value>)',
          mark: 'rgb(var(--rail-mark) / <alpha-value>)',
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
        // Variables so the look (Archivo / Atkinson Hyperlegible) and the
        // shop's appearance can switch at runtime -- values in src/index.css.
        sans: ['var(--font-sans)'],
        data: ['var(--font-data)'],
        mono: ['var(--font-mono)'],
      },
      borderRadius: {
        // Per look -- values in src/index.css.
        md: 'var(--radius-md)',
        lg: 'var(--radius-lg)',
        xl: 'var(--radius-xl)',
        '2xl': 'var(--radius-2xl)',
      },
      boxShadow: {
        // Panels lift a hair in Ticket Wallet light and sit flat on their
        // seams everywhere else; popovers always float.
        panel: 'var(--shadow-panel)',
        popover: 'var(--shadow-popover)',
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
  plugins: [],
}

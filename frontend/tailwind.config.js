/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
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
        '2xs': ['11px', { lineHeight: '16px' }],
        // Desktop body size (macOS default 13pt) -- navigation and dense UI.
        body: ['13px', { lineHeight: '20px' }],
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
      keyframes: {
        'scan-line': {
          '0%, 100%': { top: '4%' },
          '50%': { top: '92%' },
        },
      },
      animation: {
        'scan-line': 'scan-line 1.8s ease-in-out infinite',
      },
    },
  },
  plugins: [],
}

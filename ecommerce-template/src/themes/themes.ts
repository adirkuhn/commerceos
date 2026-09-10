import type { ThemeConfig } from '@/types'

// HSL triplets (no "hsl()" wrapper) -- matches how index.css's :root
// tokens are written, since Tailwind reads them as hsl(var(--x)).
const AURORA: ThemeConfig = {
  id: 'aurora', name: 'Aurora', description: 'Cool indigo, generous whitespace -- the default.',
  colors: {
    light: {
      background: '0 0% 100%', foreground: '240 10% 12%', card: '0 0% 100%', cardForeground: '240 10% 12%',
      border: '240 6% 90%', input: '240 6% 90%', ring: '243 75% 59%',
      primary: '243 75% 59%', primaryForeground: '0 0% 100%',
      secondary: '240 5% 96%', secondaryForeground: '240 10% 12%',
      muted: '240 5% 96%', mutedForeground: '240 4% 46%',
      accent: '243 75% 96%', accentForeground: '243 75% 30%',
      destructive: '0 72% 51%', destructiveForeground: '0 0% 100%',
    },
    dark: {
      background: '240 10% 7%', foreground: '240 5% 92%', card: '240 10% 10%', cardForeground: '240 5% 92%',
      border: '240 6% 20%', input: '240 6% 20%', ring: '243 75% 65%',
      primary: '243 75% 65%', primaryForeground: '240 10% 7%',
      secondary: '240 6% 16%', secondaryForeground: '240 5% 92%',
      muted: '240 6% 16%', mutedForeground: '240 5% 65%',
      accent: '243 40% 20%', accentForeground: '243 75% 85%',
      destructive: '0 63% 45%', destructiveForeground: '0 0% 100%',
    },
  },
  typography: { fontSans: 'Inter', headingWeight: '700', bodySize: '15px' },
  layout: { radius: '0.5rem', containerMaxWidth: '80rem', sectionSpacing: '5rem' },
  components: { buttonRadius: '0.5rem', cardRadius: '0.75rem', cardShadow: '0 1px 2px 0 rgb(0 0 0 / 0.05)' },
}

// Meridian -- warm, editorial, tighter radius. Proves a second theme is a
// genuinely different visual identity, not a palette swap of Aurora.
const MERIDIAN: ThemeConfig = {
  id: 'meridian', name: 'Meridian', description: 'Warm terracotta, sharp corners -- an editorial look.',
  colors: {
    light: {
      background: '32 38% 97%', foreground: '20 25% 15%', card: '30 40% 99%', cardForeground: '20 25% 15%',
      border: '25 20% 87%', input: '25 20% 87%', ring: '14 65% 45%',
      primary: '14 65% 45%', primaryForeground: '30 40% 99%',
      secondary: '30 25% 92%', secondaryForeground: '20 25% 15%',
      muted: '30 25% 92%', mutedForeground: '20 10% 45%',
      accent: '14 65% 92%', accentForeground: '14 65% 30%',
      destructive: '0 65% 48%', destructiveForeground: '0 0% 100%',
    },
    dark: {
      background: '20 20% 9%', foreground: '30 20% 92%', card: '20 18% 12%', cardForeground: '30 20% 92%',
      border: '20 15% 22%', input: '20 15% 22%', ring: '14 65% 55%',
      primary: '14 65% 55%', primaryForeground: '20 20% 9%',
      secondary: '20 15% 18%', secondaryForeground: '30 20% 92%',
      muted: '20 15% 18%', mutedForeground: '30 10% 65%',
      accent: '14 40% 22%', accentForeground: '14 65% 85%',
      destructive: '0 60% 50%', destructiveForeground: '0 0% 100%',
    },
  },
  typography: { fontSans: 'Inter', headingWeight: '800', bodySize: '15px' },
  layout: { radius: '0.125rem', containerMaxWidth: '75rem', sectionSpacing: '6rem' },
  components: { buttonRadius: '0.125rem', cardRadius: '0.125rem', cardShadow: 'none' },
}

export const THEMES: ThemeConfig[] = [AURORA, MERIDIAN]
export const DEFAULT_THEME_ID = AURORA.id

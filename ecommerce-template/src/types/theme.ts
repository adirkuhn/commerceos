// ThemeConfig controls visual identity only -- colors/typography/spacing/
// radius/component look -- never business logic. Applying a theme means
// ThemeProvider writing these values onto :root as CSS custom properties
// (see tailwind.config.js's colors, which all read from those same
// properties), so no component ever imports a theme directly.

export interface ThemeColorTokens {
  background: string
  foreground: string
  card: string
  cardForeground: string
  border: string
  input: string
  ring: string
  primary: string
  primaryForeground: string
  secondary: string
  secondaryForeground: string
  muted: string
  mutedForeground: string
  accent: string
  accentForeground: string
  destructive: string
  destructiveForeground: string
}

export interface ThemeTypography {
  fontSans: string
  headingWeight: string
  bodySize: string
}

export interface ThemeLayout {
  radius: string
  containerMaxWidth: string
  sectionSpacing: string
}

export interface ThemeComponents {
  buttonRadius: string
  cardRadius: string
  cardShadow: string
}

export interface ThemeConfig {
  id: string
  name: string
  description: string
  colors: { light: ThemeColorTokens; dark: ThemeColorTokens }
  typography: ThemeTypography
  layout: ThemeLayout
  components: ThemeComponents
}

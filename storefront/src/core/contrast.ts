// WCAG contrast helpers for theme colours: merchants pick brand colours
// freely, and the storefront derives the text colours that stay readable on
// and next to them (4.5:1 for text).

type RGB = [number, number, number]

function rgb(hex: string): RGB {
  let h = hex.replace('#', '')
  if (h.length === 3) h = [...h].map(c => c + c).join('')
  return [0, 2, 4].map(i => parseInt(h.slice(i, i + 2), 16)) as RGB
}

const toHex = (c: RGB) => '#' + c.map(v => Math.round(v).toString(16).padStart(2, '0')).join('')

function luminance(hex: string): number {
  const f = (v: number) => { v /= 255; return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4 }
  const [r, g, b] = rgb(hex)
  return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b)
}

export function contrastRatio(a: string, b: string): number {
  const [x, y] = [luminance(a), luminance(b)].sort((m, n) => n - m)
  return (x + 0.05) / (y + 0.05)
}

const WHITE = '#ffffff'
const DARK = '#111111'

/** Text colour for content on a filled background (buttons, badges). */
export function readableOn(fill: string): string {
  return contrastRatio(WHITE, fill) >= 4.5 || contrastRatio(WHITE, fill) >= contrastRatio(DARK, fill) ? WHITE : DARK
}

/** The colour itself when it reads as text on the background, otherwise
 *  the nearest darker (or lighter, on dark backgrounds) shade that does. */
export function textSafe(color: string, background: string, min = 4.5): string {
  if (contrastRatio(color, background) >= min) return color.toLowerCase()
  const towards: RGB = luminance(background) > 0.5 ? [0, 0, 0] : [255, 255, 255]
  const from = rgb(color)
  for (let t = 0.05; t <= 1; t += 0.05) {
    const mixed = toHex(from.map((v, i) => v + (towards[i] - v) * t) as RGB)
    if (contrastRatio(mixed, background) >= min) return mixed
  }
  return toHex(towards)
}

// The style settings the email editor offers: only what renders reliably
// in email clients, grouped the way a merchant thinks about a block.
export interface SectorLabels {
  text: string
  spacing: string
  size: string
  background: string
  border: string
}

// Fonts every email client has; a web font would fall back anyway.
const EMAIL_FONTS = [
  'Arial, Helvetica, sans-serif',
  'Helvetica, Arial, sans-serif',
  'Verdana, Geneva, sans-serif',
  'Tahoma, Geneva, sans-serif',
  'Trebuchet MS, Helvetica, sans-serif',
  'Georgia, serif',
  'Times New Roman, Times, serif',
  'Courier New, Courier, monospace',
]

export interface StyleSector {
  name: string
  open: boolean
  properties: Array<string | { property: string; type: string; default: string; options: Array<{ id: string; label: string }> }>
}

export function emailStyleSectors(labels: SectorLabels): StyleSector[] {
  return [
    {
      name: labels.text, open: true,
      properties: [
        { property: 'font-family', type: 'select', default: EMAIL_FONTS[0], options: EMAIL_FONTS.map(f => ({ id: f, label: f.split(',')[0] })) },
        'font-size', 'font-weight', 'color', 'line-height', 'text-align',
      ],
    },
    { name: labels.spacing, open: false, properties: ['padding', 'margin'] },
    { name: labels.size, open: false, properties: ['width', 'max-width', 'height'] },
    { name: labels.background, open: false, properties: ['background-color'] },
    { name: labels.border, open: false, properties: ['border', 'border-radius'] },
  ]
}

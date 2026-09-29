import type { SocialNetwork } from './social'

// Simple line icons (24×24, currentColor) -- no icon package in the
// shopper bundle.
const PATHS: Record<SocialNetwork, string> = {
  whatsapp: 'M4 20l1.4-4.2A8 8 0 1 1 8.3 18.7L4 20z M9 8.5c.2 2.9 3.6 6.3 6.5 6.5l1.2-1.4-2-1-1 .9c-1.2-.5-2.2-1.5-2.7-2.7l.9-1-1-2L9 8.5z',
  instagram: 'M7 3h10a4 4 0 0 1 4 4v10a4 4 0 0 1-4 4H7a4 4 0 0 1-4-4V7a4 4 0 0 1 4-4z M12 8.5a3.5 3.5 0 1 0 0 7 3.5 3.5 0 0 0 0-7z M17.5 6.5h.01',
  facebook: 'M14 21v-8h3l.5-3.5H14V7.8c0-1 .3-1.8 1.8-1.8H18V3h-2.8C12.4 3 11 4.6 11 7.4v2.1H8V13h3v8h3z',
  tiktok: 'M14 3v11.5a3.5 3.5 0 1 1-3.5-3.5 M14 3c.4 2.6 2.2 4.4 5 4.6',
  youtube: 'M3 8.2c.1-1.6 1.3-2.8 2.9-2.9C8 5.1 10 5 12 5s4 .1 6.1.3c1.6.1 2.8 1.3 2.9 2.9.1 1.3.1 2.5.1 3.8s0 2.5-.1 3.8c-.1 1.6-1.3 2.8-2.9 2.9-2.1.2-4.1.3-6.1.3s-4-.1-6.1-.3c-1.6-.1-2.8-1.3-2.9-2.9C3 14.5 3 13.3 3 12s0-2.5 0-3.8z M10 9v6l5-3-5-3z',
  x: 'M4 4l16 16 M20 4L4 20',
  maps: 'M12 21s-7-6.2-7-11.5A7 7 0 0 1 19 9.5C19 14.8 12 21 12 21z M12 7a2.5 2.5 0 1 0 0 5 2.5 2.5 0 0 0 0-5z',
  email: 'M3 6h18v12H3z M3 7l9 6 9-6',
  phone: 'M5 4h4l2 5-2.5 1.5a11 11 0 0 0 5 5L15 13l5 2v4a2 2 0 0 1-2 2A16 16 0 0 1 3 6a2 2 0 0 1 2-2z',
}

export function SocialIcon({ network, size }: { network: SocialNetwork; size: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8}
      strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" focusable="false">
      <path d={PATHS[network]} />
    </svg>
  )
}

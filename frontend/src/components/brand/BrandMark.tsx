// CommerceOS mark: a stock tag carrying barcode bars -- the thing staff
// scan and stick on product all day. Drawn in currentColor so it takes the
// colour of wherever it sits. public/favicon.svg is the same drawing; keep
// the two in sync.
export function BrandMark({ size = 16, className }: { size?: number; className?: string }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      aria-hidden="true"
      focusable="false"
      className={className}
    >
      <path
        d="M2.75 12 7.6 5.5H19.5A1.75 1.75 0 0 1 21.25 7.25v9.5A1.75 1.75 0 0 1 19.5 18.5H7.6Z"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinejoin="round"
      />
      <circle cx="7.9" cy="12" r="1.25" fill="currentColor" />
      <rect x="11.25" y="8.75" width="1.5" height="6.5" rx="0.4" fill="currentColor" />
      <rect x="14.1" y="8.75" width="2.4" height="6.5" rx="0.4" fill="currentColor" />
      <rect x="17.8" y="8.75" width="1.2" height="6.5" rx="0.4" fill="currentColor" />
    </svg>
  )
}

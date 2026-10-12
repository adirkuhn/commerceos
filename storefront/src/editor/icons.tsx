import type { ReactNode, SVGProps } from 'react'

// The few outline icons the editor chrome needs (Lucide shapes, ISC
// licence), inline so the editor bundle takes no icon package.

type IconProps = Omit<SVGProps<SVGSVGElement>, 'children'> & { size?: number }

function icon(paths: ReactNode) {
  return function Icon({ size = 16, ...rest }: IconProps) {
    return (
      <svg xmlns="http://www.w3.org/2000/svg" width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor"
        strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" aria-hidden {...rest}>
        {paths}
      </svg>
    )
  }
}

export const UndoIcon = icon(<><path d="M9 14 4 9l5-5" /><path d="M4 9h10.5a5.5 5.5 0 0 1 0 11H11" /></>)
export const RedoIcon = icon(<><path d="m15 14 5-5-5-5" /><path d="M20 9H9.5a5.5 5.5 0 0 0 0 11H13" /></>)
export const DesktopIcon = icon(<><rect width="20" height="14" x="2" y="3" rx="2" /><path d="M8 21h8M12 17v4" /></>)
export const TabletIcon = icon(<><rect width="16" height="20" x="4" y="2" rx="2" /><path d="M12 18h.01" /></>)
export const MobileIcon = icon(<><rect width="14" height="20" x="5" y="2" rx="2" /><path d="M12 18h.01" /></>)
export const ParentIcon = icon(<><path d="m14 9-5-5-5 5" /><path d="M20 20h-7a4 4 0 0 1-4-4V4" /></>)
export const CopyIcon = icon(<><rect width="14" height="14" x="8" y="8" rx="2" /><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2" /></>)
export const TrashIcon = icon(<><path d="M3 6h18M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2M10 11v6M14 11v6" /></>)
export const AddIcon = icon(<><rect width="7" height="7" x="3" y="3" rx="1" /><rect width="7" height="7" x="14" y="3" rx="1" /><rect width="7" height="7" x="14" y="14" rx="1" /><rect width="7" height="7" x="3" y="14" rx="1" /></>)
export const LayersIcon = icon(<><path d="m12.83 2.18a2 2 0 0 0-1.66 0L2.6 6.08a1 1 0 0 0 0 1.83l8.58 3.91a2 2 0 0 0 1.66 0l8.58-3.9a1 1 0 0 0 0-1.83Z" /><path d="m22 17.65-9.17 4.16a2 2 0 0 1-1.66 0L2 17.65M22 12.65l-9.17 4.16a2 2 0 0 1-1.66 0L2 12.65" /></>)
export const EyeIcon = icon(<><path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z" /><circle cx="12" cy="12" r="3" /></>)

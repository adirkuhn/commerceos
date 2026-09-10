import { forwardRef, type ComponentPropsWithoutRef, type ElementRef } from 'react'
import * as DialogPrimitive from '@radix-ui/react-dialog'
import { X } from 'lucide-react'
import { cn } from '@/lib/utils'

// A side-sheet built on the same Dialog primitive as Dialog.tsx (focus
// trap, Escape-to-close, and an accessible dialog role for free) --
// CartDrawer and the mobile nav both use this.
export const Drawer = DialogPrimitive.Root
export const DrawerTrigger = DialogPrimitive.Trigger
export const DrawerClose = DialogPrimitive.Close

const SIDE_CLASSES: Record<'left' | 'right', string> = {
  right: 'right-0 animate-slide-in-right',
  left: 'left-0',
}

export const DrawerContent = forwardRef<
  ElementRef<typeof DialogPrimitive.Content>,
  ComponentPropsWithoutRef<typeof DialogPrimitive.Content> & { side?: 'left' | 'right' }
>(({ className, children, side = 'right', ...props }, ref) => (
  <DialogPrimitive.Portal>
    <DialogPrimitive.Overlay className="fixed inset-0 z-50 bg-black/50 animate-fade-in" />
    <DialogPrimitive.Content
      ref={ref}
      className={cn(
        'fixed top-0 z-50 flex h-full w-full max-w-sm flex-col border-border bg-card shadow-lg',
        side === 'right' ? 'right-0 border-l' : 'left-0 border-r',
        SIDE_CLASSES[side],
        className,
      )}
      {...props}
    >
      {children}
      <DialogPrimitive.Close className="absolute right-4 top-4 rounded-sm opacity-70 transition-opacity hover:opacity-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
        <X className="h-4 w-4" />
        <span className="sr-only">Close</span>
      </DialogPrimitive.Close>
    </DialogPrimitive.Content>
  </DialogPrimitive.Portal>
))
DrawerContent.displayName = 'DrawerContent'

export const DrawerTitle = DialogPrimitive.Title
export const DrawerDescription = DialogPrimitive.Description

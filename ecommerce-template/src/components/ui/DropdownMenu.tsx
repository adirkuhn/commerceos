import { forwardRef, type ComponentPropsWithoutRef, type ElementRef } from 'react'
import * as DropdownPrimitive from '@radix-ui/react-dropdown-menu'
import { cn } from '@/lib/utils'

export const DropdownMenu = DropdownPrimitive.Root
export const DropdownMenuTrigger = DropdownPrimitive.Trigger

export const DropdownMenuContent = forwardRef<ElementRef<typeof DropdownPrimitive.Content>, ComponentPropsWithoutRef<typeof DropdownPrimitive.Content>>(
  ({ className, sideOffset = 6, ...props }, ref) => (
    <DropdownPrimitive.Portal>
      <DropdownPrimitive.Content
        ref={ref}
        sideOffset={sideOffset}
        className={cn(
          'z-50 min-w-[10rem] overflow-hidden rounded-[var(--radius)] border border-border bg-card p-1 text-card-foreground shadow-md animate-fade-in',
          className,
        )}
        {...props}
      />
    </DropdownPrimitive.Portal>
  ),
)
DropdownMenuContent.displayName = DropdownPrimitive.Content.displayName

export const DropdownMenuItem = forwardRef<ElementRef<typeof DropdownPrimitive.Item>, ComponentPropsWithoutRef<typeof DropdownPrimitive.Item>>(
  ({ className, ...props }, ref) => (
    <DropdownPrimitive.Item
      ref={ref}
      className={cn(
        'flex cursor-pointer select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none transition-colors focus:bg-secondary data-[disabled]:pointer-events-none data-[disabled]:opacity-50',
        className,
      )}
      {...props}
    />
  ),
)
DropdownMenuItem.displayName = DropdownPrimitive.Item.displayName

export const DropdownMenuSeparator = forwardRef<ElementRef<typeof DropdownPrimitive.Separator>, ComponentPropsWithoutRef<typeof DropdownPrimitive.Separator>>(
  ({ className, ...props }, ref) => <DropdownPrimitive.Separator ref={ref} className={cn('-mx-1 my-1 h-px bg-border', className)} {...props} />,
)
DropdownMenuSeparator.displayName = DropdownPrimitive.Separator.displayName

export const DropdownMenuLabel = forwardRef<ElementRef<typeof DropdownPrimitive.Label>, ComponentPropsWithoutRef<typeof DropdownPrimitive.Label>>(
  ({ className, ...props }, ref) => <DropdownPrimitive.Label ref={ref} className={cn('px-2 py-1.5 text-xs font-semibold text-muted-foreground', className)} {...props} />,
)
DropdownMenuLabel.displayName = DropdownPrimitive.Label.displayName

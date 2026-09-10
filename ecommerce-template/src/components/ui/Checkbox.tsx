import { forwardRef } from 'react'
import { Check } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface CheckboxProps {
  id?: string
  checked: boolean
  onCheckedChange: (checked: boolean) => void
  disabled?: boolean
  className?: string
  'aria-label'?: string
}

// A native <input type="checkbox"> under the hood (full keyboard/AT
// support for free) styled to match shadcn/ui's Checkbox look, exposing
// the same onCheckedChange(boolean) API Radix's primitive would.
export const Checkbox = forwardRef<HTMLInputElement, CheckboxProps>(
  ({ id, checked, onCheckedChange, disabled, className, ...aria }, ref) => (
    <span className={cn('relative inline-flex h-4 w-4 shrink-0 items-center justify-center', className)}>
      <input
        ref={ref}
        id={id}
        type="checkbox"
        checked={checked}
        disabled={disabled}
        onChange={(e) => onCheckedChange(e.target.checked)}
        className="peer absolute h-4 w-4 cursor-pointer appearance-none rounded-sm border border-input bg-background checked:border-primary checked:bg-primary disabled:cursor-not-allowed disabled:opacity-50"
        {...aria}
      />
      <Check className="pointer-events-none relative hidden h-3 w-3 text-primary-foreground peer-checked:block" />
    </span>
  ),
)
Checkbox.displayName = 'Checkbox'

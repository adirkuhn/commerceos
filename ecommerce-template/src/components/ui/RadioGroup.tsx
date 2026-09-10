import { createContext, useContext, type ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface RadioGroupContextValue {
  value: string
  onValueChange: (value: string) => void
  name: string
}

const RadioGroupContext = createContext<RadioGroupContextValue | null>(null)

export function RadioGroup({
  value, onValueChange, name, className, children,
}: { value: string; onValueChange: (value: string) => void; name: string; className?: string; children: ReactNode }) {
  return (
    <RadioGroupContext.Provider value={{ value, onValueChange, name }}>
      <div role="radiogroup" className={cn('grid gap-2', className)}>{children}</div>
    </RadioGroupContext.Provider>
  )
}

export function RadioGroupItem({ id, value, className }: { id: string; value: string; className?: string }) {
  const ctx = useContext(RadioGroupContext)
  if (!ctx) throw new Error('RadioGroupItem must be used within RadioGroup')
  return (
    <input
      id={id}
      type="radio"
      name={ctx.name}
      checked={ctx.value === value}
      onChange={() => ctx.onValueChange(value)}
      className={cn('h-4 w-4 cursor-pointer border-input text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring', className)}
    />
  )
}

import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

export interface DataTableColumn<T> {
  key: string
  header: string
  render: (row: T) => ReactNode
  className?: string
}

export function DataTable<T extends { id: string }>({ columns, rows, caption }: { columns: DataTableColumn<T>[]; rows: T[]; caption?: string }) {
  return (
    <div className="overflow-x-auto rounded-[var(--radius)] border border-border">
      <table className="w-full text-left text-sm">
        {caption && <caption className="sr-only">{caption}</caption>}
        <thead className="border-b border-border bg-secondary/50 text-xs uppercase tracking-wide text-muted-foreground">
          <tr>
            {columns.map(col => <th key={col.key} scope="col" className="px-4 py-3 font-medium">{col.header}</th>)}
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {rows.map(row => (
            <tr key={row.id} className="transition-colors hover:bg-secondary/30">
              {columns.map(col => <td key={col.key} className={cn('px-4 py-3', col.className)}>{col.render(row)}</td>)}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

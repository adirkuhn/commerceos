import { ArrowLeft } from 'lucide-react'

// Placeholders for a page whose record comes from the URL
// (/coupons/:id, ...): shown while the list that holds it is loading, and
// when the id matches nothing (deleted record, stale bookmark) -- instead of
// silently rendering an empty "create" form for a URL that meant "edit".

export function RecordLoading() {
  return <div className="flex h-full items-center justify-center text-sm dark:text-gray-500 text-gray-400">Loading…</div>
}

export function RecordNotFound({ label, onBack }: { label: string; onBack: () => void }) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3">
      <p className="text-sm font-medium dark:text-gray-200 text-gray-800">{label} not found</p>
      <p className="text-xs dark:text-gray-500 text-gray-400">It may have been deleted, or the link is out of date.</p>
      <button
        type="button" onClick={onBack}
        className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium dark:bg-[#1F2937] bg-gray-100 dark:text-gray-200 text-gray-700"
      >
        <ArrowLeft size={13} /> Back to list
      </button>
    </div>
  )
}

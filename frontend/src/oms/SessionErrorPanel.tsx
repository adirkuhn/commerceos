interface SessionErrorPanelProps {
  error: string
  onBack: () => void
}

// Picking/Packing main panel when a session failed to start or to load from
// its URL. Those pages otherwise render errors inside the open-session
// panel, which doesn't exist without a session -- so without this the
// error was set but never shown.
export function SessionErrorPanel({ error, onBack }: SessionErrorPanelProps) {
  return (
    <div className="flex-1 flex flex-col items-center justify-center gap-3 p-8">
      <p className="text-sm text-red-400">{error}</p>
      <button
        type="button" onClick={onBack}
        className="px-4 py-2 rounded-lg dark:bg-[#1F2937] bg-gray-100 dark:text-gray-200 text-gray-700 text-xs font-medium"
      >
        Back to orders
      </button>
    </div>
  )
}

interface Props {
  value: string
  onChange: (value: string) => void
}

export function CustomCSSPanel({ value, onChange }: Props) {
  return (
    <div className="flex h-full flex-col gap-2 p-4">
      <h3 className="text-xs font-semibold uppercase tracking-wider text-gray-500">Custom CSS</h3>
      <p className="text-[11px] text-gray-400">
        Raw CSS injected into this page on the storefront. An escape hatch for styling the canvas can&apos;t express.
      </p>
      <textarea
        aria-label="Custom CSS"
        value={value}
        onChange={e => onChange(e.target.value)}
        rows={16}
        spellCheck={false}
        className="flex-1 rounded-lg border border-gray-200 px-2.5 py-1.5 font-mono text-xs text-gray-700 outline-none focus:ring-1 focus:ring-brand-500"
      />
    </div>
  )
}

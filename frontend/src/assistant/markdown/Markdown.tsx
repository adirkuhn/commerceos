import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { isAdminPath } from './adminLinks'
import { parseBlocks, type Inline } from './parse'

interface Props {
  text: string
  // Which paths may become in-app links; everything else renders as text.
  isAllowedPath?: (path: string) => boolean
}

// Renders the assistant's answer from the markdown subset in parse.ts as
// React elements only -- no HTML is ever injected.
export function Markdown({ text, isAllowedPath = isAdminPath }: Props) {
  return (
    <div className="flex flex-col gap-2">
      {parseBlocks(text).map((block, i) => {
        if (block.kind === 'paragraph') {
          return (
            <p key={i} className="whitespace-pre-wrap break-words">
              {block.lines.map((line, j) => (
                <span key={j}>{j > 0 && <br />}{renderInline(line, isAllowedPath)}</span>
              ))}
            </p>
          )
        }
        const items = block.items.map((item, j) => <li key={j}>{renderInline(item, isAllowedPath)}</li>)
        return block.ordered
          ? <ol key={i} className="list-decimal pl-5 flex flex-col gap-0.5">{items}</ol>
          : <ul key={i} className="list-disc pl-5 flex flex-col gap-0.5">{items}</ul>
      })}
    </div>
  )
}

function renderInline(nodes: Inline[], allowed: (path: string) => boolean): ReactNode[] {
  return nodes.map((n, i) => {
    switch (n.kind) {
      case 'text':
        return n.text
      case 'code':
        return <code key={i} className="px-1 py-0.5 rounded bg-fill font-mono text-[0.95em]">{n.text}</code>
      case 'strong':
        return <strong key={i} className="font-semibold">{renderInline(n.children, allowed)}</strong>
      case 'em':
        return <em key={i}>{renderInline(n.children, allowed)}</em>
      case 'link':
        return allowed(n.href)
          ? <Link key={i} to={n.href} className="text-accent hover:underline">{renderInline(n.children, allowed)}</Link>
          : <span key={i}>{renderInline(n.children, allowed)}</span>
    }
  })
}

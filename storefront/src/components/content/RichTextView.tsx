import type { ReactNode } from 'react'
import type { RichText, RichTextInline } from '../../core/richText'
import { useRenderEnv } from '../../runtime/RenderContext'
import { storeHref } from '../../runtime/page/links'

function Inline({ span }: { span: RichTextInline }) {
  let out: ReactNode = span.text
  if (span.italic) out = <em>{out}</em>
  if (span.bold) out = <strong>{out}</strong>
  const env = useRenderEnv()
  const href = storeHref(span.href, env.storePath)
  if (href) {
    const external = /^https?:\/\//.test(href)
    out = <a href={href} {...(external ? { rel: 'noopener noreferrer', target: '_blank' } : {})}>{out}</a>
  }
  return <>{out}</>
}

const spans = (list: RichTextInline[]) => list.map((s, i) => <Inline key={i} span={s} />)

/** Renders stored rich text with React only -- no HTML ever reaches the DOM. */
export function RichTextView({ content }: { content: RichText }) {
  return (
    <>
      {content.map((block, i) => {
        if (block.type === 'paragraph') return <p key={i}>{spans(block.children ?? [])}</p>
        const List = block.type === 'numbered_list' ? 'ol' : 'ul'
        return <List key={i}>{(block.items ?? []).map((item, j) => <li key={j}>{spans(item)}</li>)}</List>
      })}
    </>
  )
}
